package students

import (
	"cloud.google.com/go/firestore"
)

const usersCollectionName = "users"

const studentAttemptsCollectionName = "student_mission_attempts"

const studentCompletionsCollectionName = "student_mission_completions"

const studentCompletionsSubcollection = "missions"

const classLearningCollectionName = "class_learning_summaries"

const maxTutorTurnsPerAttempt = 80

type FirestoreRepository struct {
	client *firestore.Client
}

func NewFirestoreRepository(client *firestore.Client) *FirestoreRepository {
	return &FirestoreRepository{
		client: client,
	}
}
