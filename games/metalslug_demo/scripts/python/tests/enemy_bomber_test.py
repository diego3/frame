"""Unit tests for enemy_bomber.py — Metal Slug demo stationary enemy that lobs bombs at the player."""

import unittest
import importlib.util
import os
from unittest.mock import MagicMock

_enemy_bomber_path = os.path.join(os.path.dirname(__file__), "..", "enemy_bomber.py")
_spec = importlib.util.spec_from_file_location(
    "enemy_bomber",
    _enemy_bomber_path
)
_eb_module = importlib.util.module_from_spec(_spec)

_eb_module.self = MagicMock()
_eb_module.engine = MagicMock()

_spec.loader.exec_module(_eb_module)
eb = _eb_module


class TestEnemyBomber(unittest.TestCase):
    """Test the fire-timing/aim rules in isolation from the engine."""

    def setUp(self):
        eb.self.reset_mock()
        eb.engine.reset_mock()
        eb._cooldown = eb.FIRE_INTERVAL

        self_positions = {"x": 100.0, "y": 200.0}
        player_positions = {"x": 400.0, "y": 380.0}
        eb.self.get_position.side_effect = lambda axis: self_positions[axis]
        eb.engine.get_entity_position.side_effect = lambda name, axis: player_positions[axis]

    def test_does_not_fire_before_cooldown_elapses(self):
        eb.update(1.0)
        eb.engine.emit.assert_not_called()

    def test_fires_once_cooldown_elapses(self):
        eb.update(eb.FIRE_INTERVAL)
        eb.engine.emit.assert_called_once()
        name, payload = eb.engine.emit.call_args[0]
        self.assertEqual(name, "SpawnEntity")
        self.assertEqual(payload["prototype"], "sphere_prototype")

    def test_spawns_at_own_position(self):
        eb.update(eb.FIRE_INTERVAL)
        _, payload = eb.engine.emit.call_args[0]
        self.assertEqual(payload["x"], 100.0)
        self.assertEqual(payload["y"], 200.0)

    def test_fuse_matches_configured_constant(self):
        eb.update(eb.FIRE_INTERVAL)
        _, payload = eb.engine.emit.call_args[0]
        self.assertEqual(payload["timer_seconds"], eb.FUSE_SECONDS)

    def test_reschedules_cooldown_after_firing(self):
        eb.update(eb.FIRE_INTERVAL)
        self.assertAlmostEqual(eb._cooldown, eb.FIRE_INTERVAL)

    def test_horizontal_velocity_aims_toward_player(self):
        """Player is to the right (x=400) of the bomber (x=100): vx must be positive, and match
        the plain constant-velocity solution (distance / flight time)."""
        eb.update(eb.FIRE_INTERVAL)
        _, payload = eb.engine.emit.call_args[0]
        expected_vx = (400.0 - 100.0) / eb.FLIGHT_TIME
        self.assertAlmostEqual(payload["vel_x"], expected_vx)

    def test_horizontal_velocity_aims_left_when_player_is_left(self):
        player_positions = {"x": -50.0, "y": 380.0}
        eb.engine.get_entity_position.side_effect = lambda name, axis: player_positions[axis]
        eb.update(eb.FIRE_INTERVAL)
        _, payload = eb.engine.emit.call_args[0]
        self.assertLess(payload["vel_x"], 0)

    def test_vertical_launch_velocity_solves_the_landing_equation(self):
        """The chosen vy must be exactly the SUVAT solution for landing on the player's y at
        t=FLIGHT_TIME -- i.e. simulating the throw by hand must land within floating-point
        tolerance of the player's y position."""
        eb.update(eb.FIRE_INTERVAL)
        _, payload = eb.engine.emit.call_args[0]
        y0 = 200.0
        landing_y = y0 + payload["vel_y"] * eb.FLIGHT_TIME + 0.5 * eb.GRAVITY * eb.FLIGHT_TIME ** 2
        self.assertAlmostEqual(landing_y, 380.0 + eb.PLAYER_FEET_OFFSET, places=6)

    def test_queries_player_by_name(self):
        eb.update(eb.FIRE_INTERVAL)
        called_names = {call.args[0] for call in eb.engine.get_entity_position.call_args_list}
        self.assertEqual(called_names, {"player"})

    def test_faces_player_to_the_right(self):
        eb.update(0.1)  # player is at x=400, bomber at x=100 -- no shot needed to check facing
        eb.self.set_facing.assert_called_with(1)

    def test_faces_player_to_the_left(self):
        player_positions = {"x": -50.0, "y": 380.0}
        eb.engine.get_entity_position.side_effect = lambda name, axis: player_positions[axis]
        eb.update(0.1)
        eb.self.set_facing.assert_called_with(-1)

    def test_faces_player_every_frame_not_only_when_firing(self):
        """Regression: facing must track the player continuously, including right from the first
        frame (this was the reported bug -- it started facing the wrong way), not just update at
        the moment of firing."""
        eb.update(0.1)
        eb.update(0.1)
        self.assertEqual(eb.self.set_facing.call_count, 2)

    def test_plays_attack_animation_when_firing(self):
        eb.update(eb.FIRE_INTERVAL)
        eb.self.play_animation.assert_any_call("attack")
        eb.self.reset_animation.assert_called_once_with("attack")

    def test_does_not_play_attack_animation_outside_of_firing(self):
        eb.update(0.1)
        eb.self.play_animation.assert_not_called()

    def test_returns_to_idle_once_attack_animation_finishes(self):
        eb.self.current_animation.return_value = "attack"
        eb.self.animation_finished.return_value = True
        eb.update(0.1)
        eb.self.play_animation.assert_called_once_with("idle")

    def test_stays_on_attack_animation_while_still_playing(self):
        eb.self.current_animation.return_value = "attack"
        eb.self.animation_finished.return_value = False
        eb.update(0.1)
        eb.self.play_animation.assert_not_called()


if __name__ == "__main__":
    unittest.main()
