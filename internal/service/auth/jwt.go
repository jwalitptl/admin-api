package auth

type JWTService struct {
	Secret string
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		Secret: secret,
	}
}
