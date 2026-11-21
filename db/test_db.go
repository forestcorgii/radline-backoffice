package db

func InitTestDB() error {
	return InitDB(":memory:")
}
