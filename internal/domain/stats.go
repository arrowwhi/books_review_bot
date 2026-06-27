package domain

type Stats struct {
	TotalBooks     int
	BooksThisYear  int
	AvgRating      float64
	GenreStats     []GenreStat
	TopBooks       []*Book
	FavoriteAspect string
	FavAspectAvg   float64
	WishlistCount  int
}

type GenreStat struct {
	Genre     Genre
	Count     int
	AvgRating float64
}
