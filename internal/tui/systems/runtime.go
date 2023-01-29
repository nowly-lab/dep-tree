package systems

import "github.com/gdamore/tcell/v2"

type ShouldQuit struct{}

func (s *ShouldQuit) Error() string {
	return "Should Quit"
}

func IsShouldQuit(err error) bool {
	_, ok := err.(*ShouldQuit)
	return ok
}

func navigate(s *State) error {
	if s.SelectedId == "" {
		return nil
	}
	err := s.Screen.Suspend()
	if err != nil {
		return err
	}
	err = s.OnNavigate(s)
	if err != nil {
		return err
	}
	err = s.Screen.Resume()
	if err != nil {
		return err
	}
	// NOTE: just to trigger an update.
	return s.Screen.PostEvent(&tcell.EventTime{})
}

func RuntimeSystem(s *State) error {
	switch ev := s.Event.(type) {
	case *tcell.EventResize:
		s.Screen.Sync()
	case *tcell.EventInterrupt:
		s.Screen.Fini()
		return nil
	case *tcell.EventKey:
		if ev.Rune() == 'q' {
			s.Screen.Fini()
			return &ShouldQuit{}
		} else if ev.Key() == tcell.KeyEnter {
			return navigate(s)
		}
	}
	return nil
}
