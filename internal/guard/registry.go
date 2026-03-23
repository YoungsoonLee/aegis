package guard

import "github.com/YoungsoonLee/aegis/internal/config"

func NewPIIGuard(cfg config.PIIGuardConfig) Guard {
	return &piiGuardAdapter{cfg: cfg}
}

func NewInjectionGuard(cfg config.InjectionGuardConfig) Guard {
	return &injectionGuardAdapter{cfg: cfg}
}

func NewContentGuard(cfg config.ContentGuardConfig) Guard {
	return &contentGuardAdapter{cfg: cfg}
}

func NewTokenGuard(cfg config.TokenGuardConfig) Guard {
	return &tokenGuardAdapter{cfg: cfg}
}
