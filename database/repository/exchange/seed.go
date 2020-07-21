package exchange

func Seed(in []Details) error {
	return InsertMany(in)
}
