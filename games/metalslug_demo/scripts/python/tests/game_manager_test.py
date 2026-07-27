"""Unit tests for game_manager.py — Metal Slug demo falling-sphere spawn rule."""

import unittest
import importlib.util
import os
from unittest.mock import MagicMock

_game_manager_path = os.path.join(os.path.dirname(__file__), "..", "game_manager.py")
_spec = importlib.util.spec_from_file_location(
    "game_manager",
    _game_manager_path
)
_gm_module = importlib.util.module_from_spec(_spec)

_gm_module.engine = MagicMock()

_spec.loader.exec_module(_gm_module)
gm = _gm_module


class TestGameManager(unittest.TestCase):
    """Test the spawn-timing/payload rules in isolation from the engine."""

    def setUp(self):
        gm.engine.reset_mock()
        gm._next_spawn_in = 1.0
        gm._spawn_count = 0

    def test_does_not_spawn_before_interval_elapses(self):
        gm.update(0.5)
        gm.engine.emit.assert_not_called()

    def test_spawns_once_interval_elapses(self):
        gm.update(1.0)
        gm.engine.emit.assert_called_once()
        name, payload = gm.engine.emit.call_args[0]
        self.assertEqual(name, "SpawnEntity")
        self.assertEqual(payload["prototype"], "sphere_prototype")

    def test_spawn_payload_has_required_keys(self):
        gm.update(1.0)
        _, payload = gm.engine.emit.call_args[0]
        for key in ("prototype", "name", "x", "y", "timer_seconds"):
            self.assertIn(key, payload)

    def test_spawn_x_within_level_bounds(self):
        gm.update(1.0)
        _, payload = gm.engine.emit.call_args[0]
        self.assertGreaterEqual(payload["x"], 0)
        self.assertLessEqual(payload["x"], gm.LEVEL_WIDTH)

    def test_spawn_y_above_level_top(self):
        gm.update(1.0)
        _, payload = gm.engine.emit.call_args[0]
        self.assertEqual(payload["y"], gm.SPAWN_Y)

    def test_fuse_within_configured_range(self):
        gm.update(1.0)
        _, payload = gm.engine.emit.call_args[0]
        self.assertGreaterEqual(payload["timer_seconds"], gm.FUSE_MIN)
        self.assertLessEqual(payload["timer_seconds"], gm.FUSE_MAX)

    def test_reschedules_next_spawn_within_configured_range(self):
        gm.update(1.0)
        self.assertGreaterEqual(gm._next_spawn_in, gm.SPAWN_INTERVAL_MIN)
        self.assertLessEqual(gm._next_spawn_in, gm.SPAWN_INTERVAL_MAX)

    def test_successive_spawns_get_distinct_names(self):
        gm.update(1.0)
        first_name = gm.engine.emit.call_args[0][1]["name"]
        gm._next_spawn_in = 0  # force an immediate second spawn
        gm.update(0.001)
        second_name = gm.engine.emit.call_args[0][1]["name"]
        self.assertNotEqual(first_name, second_name)

    def test_random_range_stays_within_bounds_over_many_samples(self):
        for _ in range(500):
            value = gm._random_range(2.0, 5.0)
            self.assertGreaterEqual(value, 2.0)
            self.assertLessEqual(value, 5.0)


if __name__ == "__main__":
    unittest.main()
