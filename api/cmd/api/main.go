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
	"classgame-api/internal/auth"
	"classgame-api/internal/bootstrap"
	"classgame-api/internal/config"
	"classgame-api/internal/modules/attachments"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/health"
	"classgame-api/internal/modules/insights"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
	"classgame-api/internal/modules/tenancy"
	"classgame-api/internal/platform/analytics"
	"classgame-api/internal/platform/firebase"
	"classgame-api/internal/platform/genai"
	"classgame-api/internal/platform/genai/redisbudget"
	"classgame-api/internal/platform/objectstorage"
	rateredis "classgame-api/internal/platform/ratelimit/redis"
	"classgame-api/internal/platform/redis"
	"classgame-api/internal/platform/redisqueue"
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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	analyticsClient, err := analytics.New(analytics.Options{
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
			analyticsClient.CaptureException(context.Background(), runErr, analytics.Properties{
				"error_source": "process",
			})
		}
		if err := analyticsClient.Close(); err != nil {
			slog.Error("analytics disconnect failed", "error", err)
		}
	}()

	startedAt := time.Now()

	firebaseConnection, err := firebase.Connect(context.Background(), firebase.Options{
		EmulatorEnabled:       cfg.FirebaseEmulatorsEnabled,
		Environment:           cfg.AppEnv,
		AuthEmulatorHost:      cfg.FirebaseAuthEmulatorHost,
		FirestoreEmulatorHost: cfg.FirestoreEmulatorHost,
		ProjectID:             cfg.FirebaseProjectID,
		CredentialsFile:       cfg.FirebaseCredentialsFile,
		CredentialsJSON:       cfg.FirebaseCredentialsJSON,
		PingTimeout:           time.Duration(cfg.FirebasePingTimeoutSeconds) * time.Second,
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
	if cfg.FirebaseBootstrapEnabled && cfg.AppRole != "worker" {
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
				Hierarchy:          tenancy.DefaultHierarchy(),
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

	redisConnection, err := redis.Connect(context.Background(), redis.Options{
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

	usersRepository := students.NewFirestoreRepository(firebaseConnection.FirestoreClient())
	providerBudget, err := redisbudget.New(redisConnection.Client(), redisbudget.Options{
		Scope: cfg.GenAISharedBudgetScope, Concurrent: int64(cfg.GenAISharedConcurrent),
		RequestsPerWindow: int64(cfg.GenAISharedRequestsPerMinute), OutputTokensPerWindow: int64(cfg.GenAISharedOutputPerMinute),
		Window: time.Minute, Lease: time.Duration(cfg.GenAISharedLeaseSeconds) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("initialize shared provider budget: %w", err)
	}
	defer providerBudget.Close()
	genAIClient, err := genai.Connect(context.Background(), genai.Options{
		Admission:       providerBudget,
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

	var storageConnection *objectstorage.Connection
	if !cfg.FirebaseEmulatorsEnabled {
		storageConnection, err = objectstorage.Connect(context.Background(), objectstorage.Options{
			BucketName:      cfg.FirebaseStorageBucket,
			CredentialsFile: cfg.FirebaseCredentialsFile,
			CredentialsJSON: cfg.FirebaseCredentialsJSON,
		})
		if err != nil {
			return fmt.Errorf("initialize mission attachment storage: %w", err)
		}
	}

	defer func() {
		if err := storageConnection.Close(); err != nil {
			slog.Error("mission attachment storage disconnect failed", "error", err)
		}
	}()

	orchestratorService := generation.NewService(generation.Dependencies{Client: genAIClient, Attachments: storageConnection})
	teacherClassRepository := classrooms.NewFirestoreRepository(firebaseConnection.FirestoreClient())
	teacherClassService := classrooms.NewService(teacherClassRepository)
	missionRepository := missions.NewFirestoreRepository(firebaseConnection.FirestoreClient())

	missionGenerationPerTeacherMinuteLimiter, err := rateredis.NewFixedWindow(
		redisConnection.Client(),
		rateredis.FixedWindowOptions{
			Limit:  missionGenerationPerTeacherMinuteLimit,
			Prefix: "classgame:mission_generation",
			Window: time.Minute,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize mission generation rate limiter: %w", err)
	}
	missionGenerationPerTeacherHourLimiter, err := rateredis.NewFixedWindow(
		redisConnection.Client(),
		rateredis.FixedWindowOptions{
			Limit:  missionGenerationPerTeacherHourLimit,
			Prefix: "classgame:mission_generation:teacher:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize hourly mission generation rate limiter: %w", err)
	}
	missionGenerationPerIPHourLimiter, err := rateredis.NewFixedWindow(
		redisConnection.Client(),
		rateredis.FixedWindowOptions{
			Limit:  missionGenerationPerIPHourLimit,
			Prefix: "classgame:mission_generation:ip:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize mission generation IP rate limiter: %w", err)
	}
	studentTutorPerStudentMinuteLimiter, err := rateredis.NewFixedWindow(
		redisConnection.Client(),
		rateredis.FixedWindowOptions{
			Limit:  studentTutorPerStudentMinuteLimit,
			Prefix: "classgame:student_tutor:student:minute",
			Window: time.Minute,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student tutor per-minute rate limiter: %w", err)
	}
	studentTutorPerStudentHourLimiter, err := rateredis.NewFixedWindow(
		redisConnection.Client(),
		rateredis.FixedWindowOptions{
			Limit:  studentTutorPerStudentHourLimit,
			Prefix: "classgame:student_tutor:student:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student tutor hourly rate limiter: %w", err)
	}
	studentTutorPerIPMinuteLimiter, err := rateredis.NewFixedWindow(
		redisConnection.Client(),
		rateredis.FixedWindowOptions{
			Limit:  studentTutorPerIPMinuteLimit,
			Prefix: "classgame:student_tutor:ip:minute",
			Window: time.Minute,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student tutor IP per-minute rate limiter: %w", err)
	}
	studentTutorPerIPHourLimiter, err := rateredis.NewFixedWindow(
		redisConnection.Client(),
		rateredis.FixedWindowOptions{
			Limit:  studentTutorPerIPHourLimit,
			Prefix: "classgame:student_tutor:ip:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student tutor IP hourly rate limiter: %w", err)
	}
	studentLearningPerClassHourLimiter, err := rateredis.NewFixedWindow(
		redisConnection.Client(),
		rateredis.FixedWindowOptions{
			Limit:  studentLearningPerClassHourLimit,
			Prefix: "classgame:student_learning:class:hour",
			Window: time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize student learning generation rate limiter: %w", err)
	}

	missionServiceOptions := []missions.ServiceOption{missions.WithAttachmentReader(storageConnection)}
	var jobQueue *redisqueue.Client
	var learningTasks students.LearningTaskEnqueuer
	if cfg.TasksEnabled {
		jobQueue, err = redisqueue.Connect(ctx, redisqueue.Options{URL: cfg.TasksRedisURL, Queue: cfg.TasksQueue, Production: cfg.AppEnv == "production"})
		if err != nil {
			return fmt.Errorf("initialize background queue: %w", err)
		}
		defer jobQueue.Close()
		learningTasks = students.NewLearningTaskScheduler(jobQueue.Destination(redisqueue.Learning), time.Duration(cfg.StudentAbandonmentSeconds)*time.Second)
		missionServiceOptions = append(missionServiceOptions, missions.WithAttachmentEnqueuer(attachments.NewTaskScheduler(jobQueue.Destination(redisqueue.Attachments), 2*time.Second)))
	}

	usersService := students.NewService(students.Dependencies{
		Recovery: usersRepository,
		Sessions: usersRepository, Learning: usersRepository, ClassLearning: usersRepository,
		Missions: missionRepository, Revisions: missionRepository, Classes: teacherClassRepository,
		TraceGenerator: orchestratorService, TutorGenerator: orchestratorService, ClassGenerator: orchestratorService,
		Tasks: learningTasks, GenerationLimiter: studentLearningPerClassHourLimiter,
		AbandonmentWindow: time.Duration(cfg.StudentAbandonmentSeconds) * time.Second,
	})
	if cfg.AppRole == "worker" || cfg.AppRole == "all" {
		copier, err := attachments.NewStorageCopier(attachments.StorageCopierOptions{BucketName: storageConnection.BucketName(), Bucket: storageConnection.Bucket()})
		if err != nil {
			return fmt.Errorf("initialize attachment copier: %w", err)
		}
		processor, err := attachments.NewProcessor(missionRepository, copier)
		if err != nil {
			return fmt.Errorf("initialize attachment processor: %w", err)
		}
		worker, err := jobQueue.Worker(redisqueue.WorkerOptions{Concurrency: cfg.TasksConcurrency, Handlers: app.BackgroundHandlers(processor, usersService)})
		if err != nil {
			return fmt.Errorf("start background worker: %w", err)
		}
		defer worker.Shutdown()
		stopRecovery := app.StartBackgroundRecovery(ctx, func(ctx context.Context, after string, limit int) (string, error) {
			report, err := usersService.RecoverPendingFinalizations(ctx, after, limit)
			slog.InfoContext(ctx, "background recovery page", "scanned", report.Scanned, "candidates", report.Candidates, "enqueued", report.Enqueued, "failed", report.Failed)
			if err != nil && report.Scanned == 0 {
				return after, err
			}
			return report.NextCursor, err
		})
		defer stopRecovery()
		slog.Info("background worker started", "queue", cfg.TasksQueue, "concurrency", cfg.TasksConcurrency)
		if cfg.AppRole == "worker" {
			<-ctx.Done()
			return nil
		}
	}

	missionService := missions.NewService(
		missionRepository,
		teacherClassService,
		orchestratorService,
		missionServiceOptions...,
	)
	dashboardRepository := insights.NewFirestoreRepository(firebaseConnection.FirestoreClient())
	dashboardService := insights.NewService(dashboardRepository, teacherClassService, missionService)
	authenticator := firebase.NewFirebaseAuthenticator(firebaseConnection.AuthClient())
	teacherAuthenticator, err := auth.RestrictProviders(authenticator, cfg.AuthTeacherProviders)
	if err != nil {
		return fmt.Errorf("initialize teacher authentication: %w", err)
	}
	studentAuthenticator, err := auth.RestrictProviders(authenticator, cfg.AuthStudentProviders)
	if err != nil {
		return fmt.Errorf("initialize student authentication: %w", err)
	}

	server := app.New(app.Options{
		Analytics: analyticsClient,
		Config:    cfg,
		Dashboard: insights.HandlerOptions{
			Authenticator:    teacherAuthenticator,
			DashboardService: dashboardService,
		},
		TeacherClass: classrooms.HandlerOptions{
			Analytics:           analyticsClient,
			Authenticator:       teacherAuthenticator,
			TeacherClassService: teacherClassService,
		},
		Mission: missions.HandlerOptions{
			AttachmentService: missionService,
			Analytics:         analyticsClient,
			Authenticator:     teacherAuthenticator,
			GenerationRateLimiters: missions.GenerationRateLimiters{
				PerIPHour:        missionGenerationPerIPHourLimiter,
				PerTeacherHour:   missionGenerationPerTeacherHourLimiter,
				PerTeacherMinute: missionGenerationPerTeacherMinuteLimiter,
			},
			MissionService: missionService,
		},
		HealthChecks: []health.Check{
			{
				Name:    health.HealthCheckServiceFirebase,
				Run:     firebaseConnection.Ping,
				Timeout: time.Duration(cfg.FirebasePingTimeoutSeconds) * time.Second,
			},
			{
				Name:    health.HealthCheckServiceRedis,
				Run:     redisConnection.Ping,
				Timeout: time.Duration(cfg.RedisPingTimeoutSeconds) * time.Second,
			},
		},
		StartedAt: startedAt,
		Users: students.HandlerOptions{
			StudentLearningService: usersService, StudentTutorService: usersService,
			StudentClassReader: usersService,
			Analytics:          analyticsClient,
			Authenticator:      studentAuthenticator,
			TutorRateLimiters: students.TutorRateLimiters{
				PerIPHour:        studentTutorPerIPHourLimiter,
				PerIPMinute:      studentTutorPerIPMinuteLimiter,
				PerStudentHour:   studentTutorPerStudentHourLimiter,
				PerStudentMinute: studentTutorPerStudentMinuteLimiter,
			},
			UsersService: usersService,
		},
	})

	if cfg.FirebaseEmulatorsEnabled {
		server.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(ctx *echo.Context) error {
				ctx.Response().Header().Set("X-ClassGame-Emulator-Project", cfg.FirebaseProjectID)
				return next(ctx)
			}
		})
	}

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
		"tasks_enabled", cfg.TasksEnabled,
		"app_role", cfg.AppRole,
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
