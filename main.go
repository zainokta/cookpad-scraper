package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/jackc/pgx/v5"
)

type Recipe struct {
	UUID      string
	RecipeID  int
	Title     string
	Link      string
	Image     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	// postgres://username:password@localhost:5432/database_name
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err.Error())
	}
	defer conn.Close(context.Background())

	if err := conn.Ping(context.Background()); err != nil {
		log.Fatal(err.Error())
	}

	recipes := make([]Recipe, 0)

	commonRecipes := []string{
		"ayam",
		"sayur",
		"ikan",
		"kue",
		"telur",
		"sapi",
		"daging",
	}
	url := "https://cookpad.com/id/cari"

	c := colly.NewCollector(
		colly.Async(),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*cookpad.*",
		Parallelism: 50,
		Delay:       1 * time.Second,
		RandomDelay: 1 * time.Second,
	})

	c.OnHTML("ul", func(e *colly.HTMLElement) {
		e.ForEach("li", func(_ int, h *colly.HTMLElement) {
			if strings.Contains(h.Attr("id"), "recipe") {
				link := h.ChildAttr("a", "href")

				title := h.ChildText("a")

				link = strings.Split(link, "/")[3]

				image := ""

				h.ForEach("picture", func(_ int, h *colly.HTMLElement) {
					imageLink := h.ChildAttr("img", "src")
					if !strings.Contains(imageLink, "avatar") {
						image = imageLink
					}
				})

				recipeID := strings.Split(link, "-")[0]

				recipeIDnum, err := strconv.Atoi(recipeID)
				if err != nil {
					log.Fatal(err)
				}

				recipes = append(recipes, Recipe{
					RecipeID:  recipeIDnum,
					Title:     title,
					Link:      link,
					Image:     image,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			}
		})
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	for _, cr := range commonRecipes {
		for i := 1; i <= 500; i++ {
			c.Visit(fmt.Sprintf("%s/%s?page=%d", url, cr, i))
		}
	}

	c.Wait()

	tx, err := conn.BeginTx(context.Background(), pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, recipe := range recipes {
		_, err = tx.Exec(context.Background(), "INSERT INTO recipes(recipe_id, title, link, image, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (recipe_id) DO UPDATE SET title = excluded.title, link = excluded.link, image = excluded.image, updated_at = now();",
			recipe.RecipeID, recipe.Title, recipe.Link, recipe.Image, time.Now(), time.Now())
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	err = tx.Commit(context.Background())
	if err != nil {
		tx.Rollback(context.Background())
		log.Fatal(err.Error())
	}
}
