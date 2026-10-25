package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"api-students/app/model"
)

var (
	ErrPortfolioNotFound = errors.New("portofolio tidak ditemukan")
	ErrProjectNotFound   = errors.New("proyek tidak ditemukan")
	ErrProjectLimit      = errors.New("jumlah proyek sudah mencapai batas")
)

const MaxProjectsPerPortfolio = 20

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	GetByStudentID(ctx context.Context, studentID int) (model.StudentPortfolio, error)
	Upsert(ctx context.Context, studentID int, headline, bio string) (model.StudentPortfolio, error)
	AddProject(ctx context.Context, studentID int, p model.Project) (model.StudentPortfolio, error)
	DeleteProject(ctx context.Context, studentID int, projectID string) (model.StudentPortfolio, error)
	GetTopTechnologies(ctx context.Context) ([]model.TechnologyStat, error)
}

type portfolioDocument struct {
	ID           bson.ObjectID            `bson:"_id,omitempty"`
	StudentID    int                      `bson:"student_id"`
	Headline     string                   `bson:"headline"`
	Bio          string                   `bson:"bio"`
	Projects     []projectDocument        `bson:"projects"`
	Certificates []certificateDocument    `bson:"certificates"`
	CreatedAt    time.Time                `bson:"created_at"`
	UpdatedAt    time.Time                `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
}

type certificateDocument struct {
	ID     bson.ObjectID `bson:"_id"`
	Name   string        `bson:"name"`
	Issuer string        `bson:"issuer"`
	Date   time.Time     `bson:"date"`
}

func (d portfolioDocument) toModel() model.StudentPortfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		techs := p.Technologies
		if techs == nil {
			techs = []string{}
		}
		projects = append(projects, model.Project{
			ID:           p.ID.Hex(),
			Title:        p.Title,
			Technologies: techs,
			Link:         p.Link,
			Year:         p.Year,
		})
	}

	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{
			ID:     c.ID.Hex(),
			Name:   c.Name,
			Issuer: c.Issuer,
			Date:   c.Date,
		})
	}

	return model.StudentPortfolio{
		ID:           d.ID.Hex(),
		StudentID:    d.StudentID,
		Headline:     d.Headline,
		Bio:          d.Bio,
		Projects:     projects,
		Certificates: certs,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

type portfolioMongoRepository struct {
	coll *mongo.Collection
}

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("student_portfolio_unique"),
		},
		{
			Keys:    bson.D{{Key: "projects.technologies", Value: 1}},
			Options: options.Index().SetName("portfolio_technologies"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func (r *portfolioMongoRepository) GetByStudentID(ctx context.Context, studentID int) (model.StudentPortfolio, error) {
	var doc portfolioDocument
	err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.StudentPortfolio{}, ErrPortfolioNotFound
		}
		return model.StudentPortfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

func (r *portfolioMongoRepository) Upsert(ctx context.Context, studentID int, headline, bio string) (model.StudentPortfolio, error) {
	t := now()
	filter := bson.M{"student_id": studentID}
	update := bson.M{
		"$set": bson.M{
			"headline":   headline,
			"bio":        bio,
			"updated_at": t,
		},
		"$setOnInsert": bson.M{
			"_id":          bson.NewObjectID(),
			"projects":     []projectDocument{},
			"certificates": []certificateDocument{},
			"created_at":   t,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var doc portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		return model.StudentPortfolio{}, fmt.Errorf("upsert portofolio: %w", err)
	}
	return doc.toModel(), nil
}

// AddProject menambahkan satu proyek secara atomik, dibatasi maksimal 20 proyek.
// Pola: "projects.19": {"$exists": false}
func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, p model.Project) (model.StudentPortfolio, error) {
	doc := projectDocument{
		ID:           bson.NewObjectID(),
		Title:        p.Title,
		Technologies: p.Technologies,
		Link:         p.Link,
		Year:         p.Year,
	}

	limitField := "projects." + strconv.Itoa(MaxProjectsPerPortfolio-1)
	filter := bson.M{
		"student_id": studentID,
		limitField:   bson.M{"$exists": false},
	}
	update := bson.M{
		"$push": bson.M{"projects": doc},
		"$set":  bson.M{"updated_at": now()},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.StudentPortfolio{}, fmt.Errorf("menambah proyek: %w", err)
	}

	// Cek apakah dokumen portofolionya ada
	count, countErr := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if countErr != nil {
		return model.StudentPortfolio{}, fmt.Errorf("memeriksa portofolio: %w", countErr)
	}
	if count == 0 {
		return model.StudentPortfolio{}, ErrPortfolioNotFound
	}
	return model.StudentPortfolio{}, ErrProjectLimit
}

func (r *portfolioMongoRepository) DeleteProject(ctx context.Context, studentID int, projectID string) (model.StudentPortfolio, error) {
	pID, err := parseID(projectID)
	if err != nil {
		return model.StudentPortfolio{}, ErrInvalidID
	}

	filter := bson.M{
		"student_id": studentID,
		"projects":   bson.M{"$elemMatch": bson.M{"_id": pID}},
	}
	update := bson.M{
		"$pull": bson.M{"projects": bson.M{"_id": pID}},
		"$set":  bson.M{"updated_at": now()},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updated portfolioDocument
	err = r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.StudentPortfolio{}, fmt.Errorf("menghapus proyek: %w", err)
	}

	count, countErr := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if countErr != nil {
		return model.StudentPortfolio{}, fmt.Errorf("memeriksa portofolio: %w", countErr)
	}
	if count == 0 {
		return model.StudentPortfolio{}, ErrPortfolioNotFound
	}
	return model.StudentPortfolio{}, ErrProjectNotFound
}

// GetTopTechnologies menjalankan aggregation pipeline dengan dua kali $unwind
// 1. $unwind: "$projects"
// 2. $unwind: "$projects.technologies"
// 3. $group: count by technology
// 4. $sort by count DESC
// 5. $limit: 10
func (r *portfolioMongoRepository) GetTopTechnologies(ctx context.Context) ([]model.TechnologyStat, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "count", Value: -1},
			{Key: "_id", Value: 1},
		}}},
		{{Key: "$limit", Value: 10}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("agregasi teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		Technology string `bson:"_id"`
		Count      int    `bson:"count"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("membaca hasil agregasi teknologi: %w", err)
	}

	stats := make([]model.TechnologyStat, 0, len(results))
	for _, item := range results {
		stats = append(stats, model.TechnologyStat{
			Technology: item.Technology,
			Count:      item.Count,
		})
	}
	return stats, nil
}
