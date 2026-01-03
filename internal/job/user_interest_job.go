package job

import (
	"Cornerstone/internal/repository"
)

type UserInterestJob struct {
	userRepo repository.UserRepo
}

func NewUserInterestJob(repo repository.UserRepo) *UserInterestJob {
	return &UserInterestJob{userRepo: repo}
}

func (s *UserInterestJob) Run() {

}
