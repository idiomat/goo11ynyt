package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/idiomat/goo11ynyt/otel/embed"
	"github.com/pgvector/pgvector-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/opentelemetry/tracing"
)

var bookPath string
var db *gorm.DB

func init() {
	flag.StringVar(&bookPath, "book", "", "path to the book ")
	flag.Parse()

	var err error
	var dsn = "postgres://postgres:password@localhost:5432/test?sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalln("failed to connect database", err)
	}
}

func main() {
	ctx := context.Background()

	// Configure a new OTLP exporter using environment variables for sending data to Honeycomb over gRPC
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %e", err)
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
	)

	// Handle shutdown to ensure all sub processes are closed correctly and telemetry is exported
	defer func() {
		_ = exp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}()

	// Register the global Tracer provider
	otel.SetTracerProvider(tp)

	// Register the W3C trace context and baggage propagators so data is propagated across services/processes
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	ctx, span := otel.Tracer("").Start(ctx, "main")
	defer span.End()

	if err := db.WithContext(ctx).Use(tracing.NewPlugin(tracing.WithoutMetrics())); err != nil {
		log.Fatalln("failed to use tracing plugin", err)
	}

	if err := db.WithContext(ctx).Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		log.Fatalln("failed to create extension", err)
	}

	if err := db.WithContext(ctx).AutoMigrate(&embed.Book{}, &embed.BookEmbedding{}); err != nil {
		log.Fatalln("failed to migrate", err)
	}

	if err := db.WithContext(ctx).Exec("CREATE INDEX ON book_embeddings USING hnsw (embedding vector_l2_ops)").Error; err != nil {
		log.Fatalln("failed to create index", err)
	}

	log.Println("Start")

	if bookPath == "" {
		log.Fatalln("book path is required")
	}

	f, err := os.Open(bookPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	var b embed.Book
	if err := db.WithContext(ctx).FirstOrCreate(&b, embed.Book{Title: "Meditations", Author: "Marcus Aurelius"}).Error; err != nil {
		log.Fatalln("failed to create book", err)
	}

	chunker, err := embed.NewChunker(
		embed.DefaultChunkSize,
		embed.DefaultChunkOverlap,
	)
	if err != nil {
		log.Fatalln(err)
	}

	httpClient := http.Client{Timeout: 30 * time.Second}

	var endpoint *url.URL
	if endpoint, err = url.Parse("http://localhost:11434/api/embeddings"); err != nil {
		log.Fatalln(err)
	}

	eg, err := embed.NewGenerator(&httpClient, endpoint, embed.DefaultModel)
	if err != nil {
		log.Fatalln(err)
	}

	chunks, err := chunker.Chunk(ctx, f)
	if err != nil {
		log.Fatalln(err)
	}

	for _, chunk := range chunks {
		vals, err := eg.Generate(ctx, chunk)
		if err != nil {
			log.Fatalln(err)
		}

		be := embed.BookEmbedding{
			BookID:    b.ID,
			Text:      chunk,
			Embedding: pgvector.NewVector(vals),
		}
		if err := db.WithContext(ctx).Save(&be).Error; err != nil {
			log.Fatalln("failed to save book embedding", err)
		}
	}

	if err := db.WithContext(ctx).Save(&b).Error; err != nil {
		log.Fatalln("failed to save book", err)
	}

	str := "How short lived the praiser and the praised, the one who remembers and the remembered."

	strEmbedding, err := eg.Generate(ctx, str)
	if err != nil {
		log.Fatalln(err)
	}

	var bookEmbeddings []embed.BookEmbedding
	db.WithContext(ctx).Clauses(
		clause.OrderBy{
			Expression: clause.Expr{
				SQL: "embedding <-> ?",
				Vars: []interface{}{
					pgvector.NewVector(strEmbedding),
				},
			},
		},
	).Limit(5).Find(&bookEmbeddings)

	for _, be := range bookEmbeddings {
		log.Println(be.Text)
	}

	log.Println("Done")
}
