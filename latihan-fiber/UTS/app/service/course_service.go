package service

import (
	"context"

	"uts-siakad/app/model"
	"uts-siakad/app/repository"
)

type CourseService interface {
	List(ctx context.Context, q model.CourseQuery) ([]model.CourseResponse, error)
}

type courseService struct {
	courseRepo repository.CourseRepository
}

func NewCourseService(courseRepo repository.CourseRepository) CourseService {
	return &courseService{courseRepo: courseRepo}
}

func (s *courseService) List(ctx context.Context, q model.CourseQuery) ([]model.CourseResponse, error) {
	return s.courseRepo.FindAll(ctx, q)
}
