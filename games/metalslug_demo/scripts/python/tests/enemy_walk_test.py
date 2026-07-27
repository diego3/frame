"""Unit tests for enemy_walk.py — Metal Slug demo enemy logic."""

import unittest
import importlib.util
import os
from unittest.mock import MagicMock

# Load enemy_walk module dynamically
_enemy_walk_path = os.path.join(os.path.dirname(__file__), "..", "enemy_walk.py")
_spec = importlib.util.spec_from_file_location(
    "enemy_walk",
    _enemy_walk_path
)
_ew_module = importlib.util.module_from_spec(_spec)

# Inject mocks before loading
_ew_module.self = MagicMock()

# Load the module
_spec.loader.exec_module(_ew_module)
ew = _ew_module


class TestEnemyWalk(unittest.TestCase):
    """Test enemy walk behavior."""

    def setUp(self):
        """Reset mocks before each test."""
        ew.self.reset_mock()

    def test_walks_left(self):
        """Enemy walks left at ENEMY_SPEED."""
        ew.update(0.016)
        ew.self.set_velocity.assert_called_once_with(-ew.ENEMY_SPEED, 0)

    def test_consistent_velocity(self):
        """Enemy maintains same velocity across frames."""
        for _ in range(10):
            ew.update(0.016)

        # All calls should be identical
        calls = ew.self.set_velocity.call_args_list
        self.assertEqual(len(calls), 10)
        for call in calls:
            self.assertEqual(call[0], (-ew.ENEMY_SPEED, 0))

    def test_speed_value(self):
        """Enemy speed is 60 game units."""
        self.assertEqual(ew.ENEMY_SPEED, 60)


if __name__ == "__main__":
    unittest.main()
