package model

import "time"

type StudentPortfolio struct {
	ID           string        `json:"id"`
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link"`
	Year         int      `json:"year"`
}

type Certificate struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Issuer string    `json:"issuer"`
	Date   time.Time `json:"date"`
}

type UpsertPortfolioRequest struct {
	Headline string `json:"headline" validate:"required,min=5,max=150"`
	Bio      string `json:"bio" validate:"required,min=10,max=1000"`
}

type AddProjectRequest struct {
	Title        string   `json:"title" validate:"required,min=3,max=120"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=10,dive,min=1,max=30"`
	Link         string   `json:"link" validate:"required,url"`
	Year         int      `json:"year" validate:"required,min=2000,max=2030"`
}

type AddCertificateRequest struct {
	Name   string `json:"name" validate:"required,min=3,max=120"`
	Issuer string `json:"issuer" validate:"required,min=3,max=120"`
	Date   string `json:"date" validate:"required,datetime=2006-01-02"`
}

type TechnologyStat struct {
	Technology string `json:"technology"`
	Count      int    `json:"count"`
}
