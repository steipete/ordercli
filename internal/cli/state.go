package cli

import (
	"errors"
	"os"

	"github.com/steipete/ordercli/internal/config"
)

type state struct {
	configPath string
	cfg        config.Config
	dirty      bool
}

func (s *state) foodora() *config.FoodoraConfig { return s.cfg.Foodora() }

func (s *state) deliveroo() *config.DeliverooConfig { return s.cfg.Deliveroo() }

func (s *state) glovo() *config.GlovoConfig { return s.cfg.Glovo() }

func (s *state) load() error {
	if s.configPath == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return err
		}
		legacy1, err := config.LegacyPathFoodcli()
		if err != nil {
			return err
		}
		legacy2, err := config.LegacyPathFoodoracli()
		if err != nil {
			return err
		}

		if _, err := os.Stat(p); err == nil {
			s.configPath = p
		} else {
			for _, legacy := range []string{legacy1, legacy2} {
				if _, err := os.Stat(legacy); err == nil {
					cfg, err := config.Load(legacy)
					if err != nil {
						return err
					}
					s.configPath = p
					s.cfg = cfg
					s.dirty = true // migrate to new path on exit
					return nil
				}
			}
			s.configPath = p
		}
	}
	cfg, err := config.Load(s.configPath)
	if err != nil {
		return err
	}
	s.cfg = cfg
	return nil
}

func (s *state) save() error {
	if !s.dirty {
		return nil
	}
	if s.configPath == "" {
		return errors.New("internal: configPath unset")
	}
	if err := config.Save(s.configPath, s.cfg); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

func (s *state) markDirty() { s.dirty = true }
