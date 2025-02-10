package geoip

type Service struct {
	config interface{} // Replace with actual config type
}

func NewService(config interface{}) *Service {
	return &Service{
		config: config,
	}
}

func (s *Service) GetLocation(ip string) (string, error) {
	// Implementation here
	return "", nil
}

func (s *Service) GetCountryCode(ip string) (string, error) {
	// Implementation here
	return "", nil
}
