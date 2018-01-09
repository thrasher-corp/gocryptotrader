package services

// Services is the overarching type across the bot package
type Services struct {
	Portfolio     *Portfolio
	Configuration *Configuration
	Websocket     *Websocket
	DefaultMain   *DefaultMain
}

// Setup sets default service variables and returns service object
func Setup() Services {
	services := Services{}
	return services
}
