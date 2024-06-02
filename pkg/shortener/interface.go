package shortener

type Shortener interface {
	GenerateShortURL(url string) (string, error)
	IsValidShortURL(url string) bool
}
