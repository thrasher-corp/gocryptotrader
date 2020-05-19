package seed

func Run() error {
	err := Exchange()
	if err != nil {
		return err
	}
	err = Currency()
	if err != nil {
		return err
	}
	return nil
}