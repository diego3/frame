"""Unit tests for explosion_effect.py — Metal Slug demo explosion VFX lifecycle."""

import unittest
import importlib.util
import os
from unittest.mock import MagicMock

_explosion_effect_path = os.path.join(os.path.dirname(__file__), "..", "explosion_effect.py")
_spec = importlib.util.spec_from_file_location(
    "explosion_effect",
    _explosion_effect_path
)
_ee_module = importlib.util.module_from_spec(_spec)

_ee_module.self = MagicMock()

_spec.loader.exec_module(_ee_module)
ee = _ee_module


class TestExplosionEffect(unittest.TestCase):
    """Test the VFX cleanup logic in isolation from the engine."""

    def setUp(self):
        ee.self.reset_mock()
        ee.self.get_timer.return_value = 1.0
        ee.self.animation_finished.return_value = False

    def test_decrements_timer_by_dt(self):
        ee.update(0.3)
        ee.self.set_timer.assert_called_once_with(0.7)

    def test_stays_alive_while_animation_is_running_and_timer_has_not_expired(self):
        ee.update(0.3)
        ee.self.destroy.assert_not_called()

    def test_destroys_when_animation_finishes(self):
        ee.self.animation_finished.return_value = True
        ee.update(0.1)
        ee.self.destroy.assert_called_once()

    def test_destroys_when_timer_expires_even_if_animation_never_reports_finished(self):
        """Defensive fallback: a misconfigured spritesheet (e.g. fps: 0) would never set
        animation_finished(), so the timer alone must still be enough to clean the entity up."""
        ee.self.get_timer.return_value = 0.05
        ee.self.animation_finished.return_value = False
        ee.update(0.1)
        ee.self.destroy.assert_called_once()

    def test_does_not_destroy_before_either_condition_is_met(self):
        ee.self.get_timer.return_value = 1.0
        ee.self.animation_finished.return_value = False
        ee.update(0.1)
        ee.self.destroy.assert_not_called()


if __name__ == "__main__":
    unittest.main()
