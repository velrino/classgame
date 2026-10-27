package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/app"
	"classgame-api/internal/bootstrap"
	"classgame-api/internal/config"
	"classgame-api/internal/modules/dashboard"
	healthcheck "classgame-api/internal/modules/health-check"
	mission "classgame-api/internal/modules/mission"
	"classgame-api/internal/modules/missionattachment"
	"classgame-api/internal/modules/orchestrator"
	teacherclass "classgame-api/internal/modules/teacher-class"
	"classgame-api/internal/modules/tenant"
	"classgame-api/internal/modules/users"
	sharedanalytics "classgame-api/internal/shared/analytics"
	sharedauth "classgame-api/internal/shared/auth"
	"classgame-api/internal/shared/cloudtasksqueue"
	sharedgoogle "classgame-api/internal/shared/google"
	sharedfirebase "classgame-api/internal/shared/google/firebase"
	sharedgenai "classgame-api/internal/shared/google/genai"
	"classgame-api/internal/shared/google/objectstorage"
	sharedratelimit "classgame-api/internal/shared/ratelimit"
	sharedredis "classgame-api/internal/shared/redis"
)

const (
	missionGenerationPerIPHourLimit        int64 = 120
	missionGenerationPerTeacherHourLimit   int64 = 30
	missionGenerationPerTeacherMinuteLimit int64 = 6
	studentLearningPerClassHourLimit       int64 = 180
	studentTutorPerIPHourLimit             int64 = 900
	studentTutorPerIPMinuteLimit           int64 = 180
	studentTutorPerStudentHourLimit        int64 = 60
	studentTutorPerStudentMinuteLimit      int64 = 12
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	analyticsClient, err := sharedanalytics.New(sharedanalytics.Options{
		Enabled:      cfg.PostHogEnabled,
		Environment:  cfg.AppEnv,
		Host:         cfg.PostHogHost,
		ProjectToken: cfg.PostHogProjectToken,
	})
	if err != nil {
		return fmt.Errorf("initialize analytics: %w", err)
	}
	defer func() {
		if runErr != nil {
			analyticsClient.CaptureException(context.Background(), runErr, sharedanalytics.Properties{
				"error_source": "process",
			})
		}
		if err := analyticsClient.Close(); err != nil {
			slog.Error("analytics disconnect failed", "error", err)
		}
	}()

	startedAt := time.Now()

	firebaseConnection, err := sharedgoogle.Connect(context.Background(), sharedgoogle.Options{
		ProjectID:       cfg.FirebaseProjectID,
		CredentialsFile: cfg.FirebaseCredentialsFile,
		CredentialsJSON: cfg.FirebaseCredentialsJSON,
		PingTimeout:     time.Duration(cfg.FirebasePingTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("initialize firebase: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := firebaseConnection.Disconnect(ctx); err != nil {
			slog.Error("firebase disconnect failed", "error", err)
		}
	}()
	if cfg.FirebaseBootstrapEnabled {
		bootstrapContext, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(cfg.FirebaseBootstrapTimeoutSeconds)*time.Second,
		)
		bootstrapResult, bootstrapErr := bootstrap.Ensure(
			bootstrapContext,
			firebaseConnection.AuthClient(),
			firebaseConnection.FirestoreClient(),
			bootstrap.Options{
				TeacherEmail:       cfg.DemoTeacherEmail,
				TeacherPassword:    cfg.DemoTeacherPassword,
				TeacherDisplayName: cfg.DemoTeacherDisplayName,
				Hierarchy:          tenant.DefaultHierarchy(),
			},
		)
		cancel()
		if bootstrapErr != nil {
			return fmt.Errorf("bootstrap firebase data: %w", bootstrapErr)
		}
		slog.Info("firebase bootstrap complete",
			"teacher_uid", bootstrapResult.TeacherUID,
			"teacher_created", bootstrapResult.TeacherCreated,
			"tenant_created", bootstrapResult.TenantCreated,
			"district_created", bootstrapResult.DistrictCreated,
			"school_created", bootstrapResult.SchoolCreated,
		)
	}

	redisConnection, err := sharedredis.Connect(context.Background(), sharedredis.Options{
		URL:         cfg.RedisURL,
		CACertPath:  cfg.RedisCACertPath,
		CACertPEM:   cfg.RedisCACertPEM,
		PingTimeout: time.Duration(cfg.RedisPingTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("initialize redis: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := redisConnection.Disconnect(ctx); err != nil {
			slog.Error("redis disconnect failed", "error", err)
		}
	}()

	usersRepository := users.NewRepository(firebaseConnection.FirestoreClient())
	genAIClient, err := sharedgenai.Connect(context.Background(), sharedgenai.Options{
		Enabled:         cfg.GenAIEnabled,
		Backend:         cfg.GenAIBackend,
		Model:           cfg.GenAIModel,
		APIKey:          cfg.GenAIAPIKey,
		ProjectID:       cfg.GenAIProjectID,
		Location:        cfg.GenAILocation,
		CredentialsFile: cfg.GenAICredentialsFile,
		CredentialsJSON: cfg.GenAICredentialsJSON,
		RequestTimeout:  time.Duration(cfg.GenAIRequestTimeoutSeconds) * time.Second,
		MaxOutputTokens: int32(cfg.GenAIMaxOutputTokens),
	})
	if err != nil {
		return fmt.Errorf("initialize genai: %w", err)
	}

	storageConnection, err := objectstorage.Connect(context.Background(), objectstorage.Options{
		BucketName:      cfg.FirebaseStorageBucket,
		CredentialsFile: cfg.FirebaseCredentialsFile,
		CredentialsJSON: cfg.FirebaseCredentialsJSON,
	})
	if err != nil {
		return fmt.Errorf("initialize mission attachment storage: %w", err)
	}
	defer func() {
		if err := storageConnection.Close(); err != nil {
			slog.Error("mission attachment storage disconnect failed", "error", err)
		}
	}()

	orchestratorService := orchestrator.NewService(genAIClient, storageConnection)
	teacherClassRepository := teacherclass.NewRepository(firebaseConnection.FirestoreClient())
	teacherClassService := teacherclass.NewService(teacherClassRepository)
	missionRepository := mission.NewRepository(firebaseConnection.FirestoreClient())
	usersService := users.NewService(usersRepository, missionRepository, orchestratorService)
	usersService.ConfigureClassReader(teacherClassRepository)

	missionGenerationPerTeacherMinuteLimiter, err := sharedratelimit.NewRedisFixedWindow(
		redisConnection.Client(),
		sharedratelimit.FixedWindowOptions{
			Limit:  missionGenerationPerTeacherMinuteLimit,
			Prefix: "classgame:mission_generation",
			Window: time.Minute,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize mission generation rate limiter: %w", err)
	}
	missionGenerationPerTeacherHourLimiter, err := sharedratelimit.NewRedisFixedWindow(
		redisConnection.Client(),
		sharedratelimit.FixedWindowOptions{
			Limit:  missionGenerationPerTeacherHourLimit,
			Prefix: "classgame:mission_generation:teacher:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize hourly mission generation rate limiter: %w", err)
	}
	missionGenerationPerIPHourLimiter, err := sharedratelimit.NewRedisFixedWindow(
		redisConnection.Client(),
		sharedratelimit.FixedWindowOptions{
			Limit:  missionGenerationPerIPHourLimit,
			Prefix: "classgame:mission_generation:ip:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize mission generation IP rate limiter: %w", err)
	}
	studentTutorPerStudentMinuteLimiter, err := sharedratelimit.NewRedisFixedWindow(
		redisConnection.Client(),
		sharedratelimit.FixedWindowOptions{
			Limit:  studentTutorPerStudentMinuteLimit,
			Prefix: "classgame:student_tutor:student:minute",
			Window: time.Minute,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student tutor per-minute rate limiter: %w", err)
	}
	studentTutorPerStudentHourLimiter, err := sharedratelimit.NewRedisFixedWindow(
		redisConnection.Client(),
		sharedratelimit.FixedWindowOptions{
			Limit:  studentTutorPerStudentHourLimit,
			Prefix: "classgame:student_tutor:student:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student tutor hourly rate limiter: %w", err)
	}
	studentTutorPerIPMinuteLimiter, err := sharedratelimit.NewRedisFixedWindow(
		redisConnection.Client(),
		sharedratelimit.FixedWindowOptions{
			Limit:  studentTutorPerIPMinuteLimit,
			Prefix: "classgame:student_tutor:ip:minute",
			Window: time.Minute,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student tutor IP per-minute rate limiter: %w", err)
	}
	studentTutorPerIPHourLimiter, err := sharedratelimit.NewRedisFixedWindow(
		redisConnection.Client(),
		sharedratelimit.FixedWindowOptions{
			Limit:  studentTutorPerIPHourLimit,
			Prefix: "classgame:student_tutor:ip:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student tutor IP hourly rate limiter: %w", err)
	}
	studentLearningPerClassHourLimiter, err := sharedratelimit.NewRedisFixedWindow(
		redisConnection.Client(),
		sharedratelimit.FixedWindowOptions{
			Limit:  studentLearningPerClassHourLimit,
			Prefix: "classgame:student_learning:class:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student learning generation rate limiter: %w", err)
	}
	usersService.ConfigureLearningGenerationLimiter(studentLearningPerClassHourLimiter)

	missionServiceOptions := []mission.ServiceOption{mission.WithAttachmentReader(storageConnection)}
	var missionAttachmentHandler http.Handler
	var studentLearningTaskHandler http.Handler
	if cfg.CloudTasksEnabled {

		storageCopier, err := missionattachment.NewStorageCopier(missionattachment.StorageCopierOptions{
			BucketName: storageConnection.BucketName(),
			Bucket:     storageConnection.Bucket(),
		})
		if err != nil {
			return fmt.Errorf("initialize mission attachment copier: %w", err)
		}

		taskQueue, err := cloudtasksqueue.Connect(context.Background(), cloudtasksqueue.Options{
			ProjectID:          cfg.CloudTasksProjectID,
			Location:           cfg.CloudTasksLocation,
			Queue:              cfg.CloudTasksQueue,
			TargetURL:          cfg.CloudTasksTargetURL,
			OIDCServiceAccount: cfg.CloudTasksServiceAccount,
			OIDCAudience:       cfg.CloudTasksOIDCAudience,
			DispatchDeadline:   290 * time.Second,
		})
		if err != nil {
			return fmt.Errorf("initialize mission attachment task queue: %w", err)
		}
		defer func() {
			if err := taskQueue.Close(); err != nil {
				slog.Error("mission attachment task queue disconnect failed", "error", err)
			}
		}()

		learningTaskQueue, err := cloudtasksqueue.Connect(context.Background(), cloudtasksqueue.Options{
			ProjectID:          cfg.CloudTasksProjectID,
			Location:           cfg.CloudTasksLocation,
			Queue:              cfg.CloudTasksQueue,
			TargetURL:          cfg.CloudTasksLearningTargetURL,
			OIDCServiceAccount: cfg.CloudTasksServiceAccount,
			OIDCAudience:       cfg.CloudTasksOIDCAudience,
			DispatchDeadline:   290 * time.Second,
		})
		if err != nil {
			return fmt.Errorf("initialize student learning task queue: %w", err)
		}
		defer func() {
			if err := learningTaskQueue.Close(); err != nil {
				slog.Error("student learning task queue disconnect failed", "error", err)
			}
		}()
		usersService.ConfigureLearningTasks(
			users.NewCloudLearningTaskEnqueuer(
				learningTaskQueue,
				time.Duration(cfg.StudentAbandonmentSeconds)*time.Second,
			),
			time.Duration(cfg.StudentAbandonmentSeconds)*time.Second,
		)

		attachmentEnqueuer := missionattachment.NewCloudTaskEnqueuer(taskQueue, 2*time.Second)
		missionServiceOptions = append(
			missionServiceOptions,
			mission.WithAttachmentEnqueuer(attachmentEnqueuer),
		)

		attachmentProcessor, err := missionattachment.NewProcessor(missionRepository, storageCopier)
		if err != nil {
			return fmt.Errorf("initialize mission attachment processor: %w", err)
		}
		attachmentAuthenticator, err := missionattachment.NewOIDCAuthenticator(
			context.Background(),
			cfg.CloudTasksOIDCAudience,
			cfg.CloudTasksServiceAccount,
		)
		if err != nil {
			return fmt.Errorf("initialize mission attachment task authentication: %w", err)
		}
		missionAttachmentHandler, err = missionattachment.NewHTTPHandler(
			attachmentAuthenticator,
			attachmentProcessor,
		)
		if err != nil {
			return fmt.Errorf("initialize mission attachment task handler: %w", err)
		}
		studentLearningTaskHandler, err = users.NewLearningTaskHTTPHandler(
			attachmentAuthenticator,
			usersService,
		)
		if err != nil {
			return fmt.Errorf("initialize student learning task handler: %w", err)
		}
	}

	missionService := mission.NewService(
		missionRepository,
		teacherClassService,
		orchestratorService,
		missionServiceOptions...,
	)
	dashboardRepository := dashboard.NewRepository(firebaseConnection.FirestoreClient())
	dashboardService := dashboard.NewService(dashboardRepository, teacherClassService, missionService)
	authenticator := sharedfirebase.NewFirebaseAuthenticator(firebaseConnection.AuthClient())
	teacherAuthenticator, err := sharedauth.RestrictProviders(authenticator, cfg.AuthTeacherProviders)
	if err != nil {
		return fmt.Errorf("initialize teacher authentication: %w", err)
	}
	studentAuthenticator, err := sharedauth.RestrictProviders(authenticator, cfg.AuthStudentProviders)
	if err != nil {
		return fmt.Errorf("initialize student authentication: %w", err)
	}

	server := app.New(app.Options{
		Analytics: analyticsClient,
		Config:    cfg,
		Dashboard: dashboard.HandlerOptions{
			Authenticator:    teacherAuthenticator,
			DashboardService: dashboardService,
		},
		TeacherClass: teacherclass.HandlerOptions{
			Analytics:           analyticsClient,
			Authenticator:       teacherAuthenticator,
			TeacherClassService: teacherClassService,
		},
		Mission: mission.HandlerOptions{
			AttachmentService: missionService,
			Analytics:         analyticsClient,
			Authenticator:     teacherAuthenticator,
			GenerationRateLimiters: mission.GenerationRateLimiters{
				PerIPHour:        missionGenerationPerIPHourLimiter,
				PerTeacherHour:   missionGenerationPerTeacherHourLimiter,
				PerTeacherMinute: missionGenerationPerTeacherMinuteLimiter,
			},
			MissionService: missionService,
		},
		MissionAttachment:   missionAttachmentHandler,
		StudentLearningTask: studentLearningTaskHandler,
		HealthChecks: []healthcheck.Check{
			{
				Name:    healthcheck.HealthCheckServiceFirebase,
				Run:     firebaseConnection.Ping,
				Timeout: time.Duration(cfg.FirebasePingTimeoutSeconds) * time.Second,
			},
			{
				Name:    healthcheck.HealthCheckServiceRedis,
				Run:     redisConnection.Ping,
				Timeout: time.Duration(cfg.RedisPingTimeoutSeconds) * time.Second,
			},
		},
		StartedAt: startedAt,
		Users: users.HandlerOptions{
			StudentClassReader: usersService,
			Analytics:          analyticsClient,
			Authenticator:      studentAuthenticator,
			TutorRateLimiters: users.TutorRateLimiters{
				PerIPHour:        studentTutorPerIPHourLimiter,
				PerIPMinute:      studentTutorPerIPMinuteLimiter,
				PerStudentHour:   studentTutorPerStudentHourLimiter,
				PerStudentMinute: studentTutorPerStudentMinuteLimiter,
			},
			UsersService: usersService,
		},
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("server started",
		"host", cfg.Host,
		"port", cfg.Port,
		"app_env", cfg.AppEnv,
		"firebase_enabled", firebaseConnection.IsEnabled(),
		"firebase_project_id", firebaseConnection.ProjectID(),
		"genai_enabled", genAIClient.IsEnabled(),
		"genai_backend", cfg.GenAIBackend,
		"genai_model", cfg.GenAIModel,
		"redis_enabled", redisConnection.IsEnabled(),
		"cloud_tasks_enabled", cfg.CloudTasksEnabled,
		"posthog_enabled", analyticsClient.Enabled(),
		"firebase_storage_bucket", cfg.FirebaseStorageBucket,
		"last_deploy", startedAt.Format(time.RFC3339Nano),
	)

	startConfig := echo.StartConfig{
		Address:         cfg.Address(),
		HideBanner:      true,
		HidePort:        true,
		GracefulTimeout: 10 * time.Second,
		OnShutdownError: func(err error) {
			slog.Error("server shutdown failed", "error", err)
		},
	}

	if err := startConfig.Start(ctx, server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("start server: %w", err)
	}

	slog.Info("server stopped")

	return nil
}
