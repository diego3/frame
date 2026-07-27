"""Unit tests for sphere_timer.py — Metal Slug demo falling-sphere countdown logic."""

import unittest
import importlib.util
import os
from unittest.mock import MagicMock

_sphere_timer_path = os.path.join(os.path.dirname(__file__), "..", "sphere_timer.py")
_spec = importlib.util.spec_from_file_location(
    "sphere_timer",
    _sphere_timer_path
)
_st_module = importlib.util.module_from_spec(_spec)

_st_module.self = MagicMock()
_st_module.engine = MagicMock()

_spec.loader.exec_module(_st_module)
st = _st_module


class TestSphereTimer(unittest.TestCase):
    """Test the countdown/explode logic in isolation from the engine."""

    def setUp(self):
        st.self.reset_mock()
        st.engine.reset_mock()
        st.self.get_name.return_value = "sphere_1"
        st.self.get_position.side_effect = lambda axis: {"x": 111.0, "y": 222.0}[axis]

    def _set_remaining(self, seconds):
        st.self.get_timer.return_value = seconds

    def test_decrements_timer_by_dt(self):
        """update(dt) writes back remaining - dt."""
        self._set_remaining(4.0)
        st.update(0.5)
        st.self.set_timer.assert_called_once_with(3.5)

    def test_does_not_destroy_while_time_remains(self):
        self._set_remaining(4.0)
        st.update(0.5)
        st.self.destroy.assert_not_called()

    def test_logs_countdown_on_whole_second_crossing(self):
        """Crossing from 3.1s to 2.9s should log the newly-reached second, 3."""
        self._set_remaining(3.1)
        with self._capture_stdout() as out:
            st.update(0.2)
        self.assertIn("sphere_1: 3", out.getvalue())

    def test_does_not_log_when_no_whole_second_is_crossed(self):
        """Both before and after remain within the same whole second (5 -> 5th)."""
        self._set_remaining(5.0)
        with self._capture_stdout() as out:
            st.update(0.1)
        self.assertEqual(out.getvalue(), "")

    def test_explodes_and_logs_when_timer_reaches_zero(self):
        self._set_remaining(0.1)
        with self._capture_stdout() as out:
            st.update(0.2)
        st.self.destroy.assert_called_once()
        self.assertIn("sphere_1: exploded", out.getvalue())

    def test_explodes_exactly_at_zero(self):
        self._set_remaining(0.2)
        st.update(0.2)
        st.self.destroy.assert_called_once()

    def test_spawns_explosion_vfx_at_own_position_when_exploding(self):
        self._set_remaining(0.1)
        st.update(0.2)
        st.engine.emit.assert_called_once()
        name, payload = st.engine.emit.call_args[0]
        self.assertEqual(name, "SpawnEntity")
        self.assertEqual(payload["prototype"], "explosion_prototype")
        self.assertEqual(payload["x"], 111.0)
        self.assertEqual(payload["y"], 222.0)

    def test_does_not_spawn_explosion_vfx_while_time_remains(self):
        self._set_remaining(4.0)
        st.update(0.5)
        st.engine.emit.assert_not_called()

    def test_explosion_vfx_spawned_before_destroy(self):
        """The explosion should be visible even though the sphere itself is destroyed the same
        frame -- spawning it first (before self.destroy()) ensures MainMenu.spawnEntity sees a
        still-consistent world regardless of destroy-processing order."""
        self._set_remaining(0.1)
        call_order = []
        st.engine.emit.side_effect = lambda *a, **k: call_order.append("emit")
        st.self.destroy.side_effect = lambda: call_order.append("destroy")
        st.update(0.2)
        self.assertEqual(call_order, ["emit", "destroy"])

    def test_uses_entity_name_in_log_messages(self):
        """Different spheres (different self.get_name()) must not share countdown state --
        each update() call reads remaining directly from that entity's own Timer component via
        self.get_timer(), so two concurrent spheres never interfere with each other."""
        st.self.get_name.return_value = "sphere_7"
        self._set_remaining(1.1)
        with self._capture_stdout() as out:
            st.update(0.2)
        self.assertIn("sphere_7: 1", out.getvalue())

    def test_does_not_double_log_across_multiple_small_steps_within_same_second(self):
        self._set_remaining(3.05)
        with self._capture_stdout() as out:
            st.update(0.03)  # 3.05 -> 3.02, still within second 4
        self.assertEqual(out.getvalue(), "")

    @staticmethod
    def _capture_stdout():
        import io
        import contextlib
        return contextlib.redirect_stdout(io.StringIO())


if __name__ == "__main__":
    unittest.main()
