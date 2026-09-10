package recommendations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/victor141516/Nestrr/internal/store"
)

type tmdbNamedID struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	LogoPath      string  `json:"logo_path"`
	OriginCountry string  `json:"origin_country"`
	ProfilePath   string  `json:"profile_path"`
	Department    string  `json:"known_for_department"`
	Popularity    float64 `json:"popularity"`
	Adult         bool    `json:"adult"`
}

type tmdbCredit struct {
	tmdbNamedID
	CreditID   string `json:"credit_id"`
	Character  string `json:"character"`
	Order      int    `json:"order"`
	Job        string `json:"job"`
	Department string `json:"department"`
}

type tmdbKeywords struct {
	Keywords []tmdbNamedID `json:"keywords"`
	Results  []tmdbNamedID `json:"results"`
}

type completeMediaDocument struct {
	ID                  int           `json:"id"`
	Title               string        `json:"title"`
	Name                string        `json:"name"`
	OriginalTitle       string        `json:"original_title"`
	OriginalName        string        `json:"original_name"`
	Overview            string        `json:"overview"`
	PosterPath          string        `json:"poster_path"`
	BackdropPath        string        `json:"backdrop_path"`
	ReleaseDate         string        `json:"release_date"`
	FirstAirDate        string        `json:"first_air_date"`
	VoteAverage         float64       `json:"vote_average"`
	VoteCount           int           `json:"vote_count"`
	Popularity          float64       `json:"popularity"`
	Genres              []tmdbNamedID `json:"genres"`
	ProductionCompanies []tmdbNamedID `json:"production_companies"`
	CreatedBy           []tmdbNamedID `json:"created_by"`
	Keywords            tmdbKeywords  `json:"keywords"`
	Credits             struct {
		Cast []tmdbCredit `json:"cast"`
		Crew []tmdbCredit `json:"crew"`
	} `json:"credits"`
}

type parsedMedia struct {
	Catalog   store.CatalogMedia
	People    []store.Person
	Credits   []store.MediaCredit
	Companies []store.Company
	Genres    []store.Genre
	Keywords  []store.Keyword
}

func parseCompleteMedia(mediaType string, expectedID int, payload []byte) (parsedMedia, error) {
	var document completeMediaDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		return parsedMedia{}, fmt.Errorf("decode complete TMDB %s document: %w", mediaType, err)
	}
	if document.ID <= 0 || document.ID != expectedID {
		return parsedMedia{}, fmt.Errorf("complete TMDB %s document has unexpected id %d", mediaType, document.ID)
	}
	if mediaType != "movie" && mediaType != "tv" {
		return parsedMedia{}, fmt.Errorf("unsupported media type %q", mediaType)
	}
	title, originalTitle, releaseDate := document.Title, document.OriginalTitle, document.ReleaseDate
	if mediaType == "tv" {
		title, originalTitle, releaseDate = document.Name, document.OriginalName, document.FirstAirDate
	}
	if strings.TrimSpace(title) == "" {
		return parsedMedia{}, fmt.Errorf("complete TMDB %s document %d has no title", mediaType, document.ID)
	}

	features := store.CatalogFeatures{}
	genres := make([]store.Genre, 0, len(document.Genres))
	for _, genre := range document.Genres {
		features.Genres = appendUniqueID(features.Genres, genre.ID)
		if genre.ID > 0 {
			genres = append(genres, store.Genre{ID: genre.ID, Name: genre.Name})
		}
	}
	keywords := document.Keywords.Keywords
	if mediaType == "tv" {
		keywords = document.Keywords.Results
	}
	parsedKeywords := make([]store.Keyword, 0, len(keywords))
	for _, keyword := range keywords {
		features.Keywords = appendUniqueID(features.Keywords, keyword.ID)
		if keyword.ID > 0 {
			parsedKeywords = append(parsedKeywords, store.Keyword{ID: keyword.ID, Name: keyword.Name})
		}
	}

	companies := make([]store.Company, 0, len(document.ProductionCompanies))
	for _, company := range document.ProductionCompanies {
		if company.ID <= 0 {
			continue
		}
		features.Companies = appendUniqueID(features.Companies, company.ID)
		companies = append(companies, store.Company{TMDBID: company.ID, Name: company.Name, LogoPath: company.LogoPath, OriginCountry: company.OriginCountry})
	}

	peopleByID := map[int]store.Person{}
	credits := make([]store.MediaCredit, 0, len(document.Credits.Cast)+len(document.Credits.Crew)+len(document.CreatedBy))
	for _, cast := range document.Credits.Cast {
		if cast.ID <= 0 {
			continue
		}
		peopleByID[cast.ID] = personFromTMDB(cast.tmdbNamedID)
		if cast.Order < 20 {
			features.Cast = appendUniqueID(features.Cast, cast.ID)
		}
		credits = append(credits, store.MediaCredit{MediaType: mediaType, MediaTMDBID: document.ID, CreditKey: creditKey("cast", cast), PersonTMDBID: cast.ID, CreditKind: "cast", Department: cast.Department, Character: cast.Character, Order: cast.Order})
	}
	for _, crew := range document.Credits.Crew {
		if crew.ID <= 0 {
			continue
		}
		peopleByID[crew.ID] = personFromTMDB(crew.tmdbNamedID)
		if strings.EqualFold(crew.Job, "Director") {
			features.Directors = appendUniqueID(features.Directors, crew.ID)
		}
		credits = append(credits, store.MediaCredit{MediaType: mediaType, MediaTMDBID: document.ID, CreditKey: creditKey("crew", crew), PersonTMDBID: crew.ID, CreditKind: "crew", Department: crew.Department, Job: crew.Job, Order: crew.Order})
	}
	for index, creator := range document.CreatedBy {
		if creator.ID <= 0 {
			continue
		}
		peopleByID[creator.ID] = personFromTMDB(creator)
		features.Directors = appendUniqueID(features.Directors, creator.ID)
		credits = append(credits, store.MediaCredit{MediaType: mediaType, MediaTMDBID: document.ID, CreditKey: fmt.Sprintf("creator:%d", creator.ID), PersonTMDBID: creator.ID, CreditKind: "crew", Department: "Production", Job: "Creator", Order: index})
	}

	people := make([]store.Person, 0, len(peopleByID))
	for _, person := range peopleByID {
		people = append(people, person)
	}
	featurePayload, err := json.Marshal(features)
	if err != nil {
		return parsedMedia{}, err
	}
	return parsedMedia{
		Catalog: store.CatalogMedia{
			CatalogKey: store.CatalogKey{MediaType: mediaType, TMDBID: document.ID},
			Payload:    payload, Features: featurePayload, Title: title, OriginalTitle: originalTitle,
			Overview: document.Overview, PosterPath: document.PosterPath, BackdropPath: document.BackdropPath,
			ReleaseDate: releaseDate, VoteAverage: document.VoteAverage, VoteCount: document.VoteCount, Popularity: document.Popularity,
		},
		People: people, Credits: credits, Companies: companies, Genres: genres, Keywords: parsedKeywords,
	}, nil
}

func personFromTMDB(person tmdbNamedID) store.Person {
	return store.Person{TMDBID: person.ID, Name: person.Name, ProfilePath: person.ProfilePath, KnownForDepartment: person.Department, Popularity: person.Popularity, Adult: person.Adult}
}

func creditKey(kind string, credit tmdbCredit) string {
	if strings.TrimSpace(credit.CreditID) != "" {
		return credit.CreditID
	}
	return fmt.Sprintf("%s:%d:%s:%d", kind, credit.ID, credit.Job, credit.Order)
}

func appendUniqueID(values []int, id int) []int {
	if id <= 0 {
		return values
	}
	for _, existing := range values {
		if existing == id {
			return values
		}
	}
	return append(values, id)
}
