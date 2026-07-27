"""Unit tests for player_controller.py — Metal Slug demo player logic."""

import unittest
import importlib.util
import sys
import os
from unittest.mock import MagicMock

# Load player_controller module dynamically (can't use normal import due to global state)
_player_controller_path = os.path.join(os.path.dirname(__file__), "..", "player_controller.py")
_spec = importlib.util.spec_from_file_location(
    "player_controller",
    _player_controller_path
)
_pc_module = importlib.util.module_from_spec(_spec)

# Inject mocks before loading
_pc_module.self = MagicMock()
_pc_module.engine = MagicMock()

# Load the module
_spec.loader.exec_module(_pc_module)
pc = _pc_module


class TestPlayerController(unittest.TestCase):
    """Test player controller state and behavior."""

    def setUp(self):
        """Reset global state before each test."""
        pc.facing_x = 1.0
        pc.pending_shoot = False
        pc.pending_jump = False
        pc.is_grounded = True
        pc.self.reset_mock()
        pc.engine.reset_mock()
        pc.self.get_intent.return_value = 0  # default: no movement
        pc.self.get_velocity.return_value = 0  # default: at rest vertically

    def test_initial_state(self):
        """Player starts grounded, facing right."""
        self.assertTrue(pc.is_grounded)
        self.assertEqual(pc.facing_x, 1.0)

    def test_vertical_velocity_left_untouched(self):
        """update() reads current vertical velocity and passes it through unchanged (Box2D
        integrates gravity for dynamic bodies on its own -- the script must not fight it)."""
        pc.self.get_velocity.return_value = 123.4
        pc.update(0.016)
        pc.self.set_velocity.assert_called_once()
        call_args = pc.self.set_velocity.call_args[0]
        self.assertEqual(call_args[1], 123.4)

    def test_ground_detection_via_begin_contact(self):
        """BeginContact between player and ground sets is_grounded."""
        pc.is_grounded = False
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertTrue(pc.is_grounded)

    def test_ground_detection_via_begin_contact_name_order_reversed(self):
        """BeginContact sets is_grounded regardless of which name is A vs B."""
        pc.is_grounded = False
        pc.on_event("BeginContact", {"GameObjectNameA": "ground", "GameObjectNameB": "player"})
        self.assertTrue(pc.is_grounded)

    def test_ground_detection_via_crate_contact(self):
        """BeginContact against a crate also counts as grounded."""
        pc.is_grounded = False
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "crate_2"})
        self.assertTrue(pc.is_grounded)

    def test_ground_detection_ignores_unrelated_contact(self):
        """BeginContact between unrelated objects does not affect the player."""
        pc.is_grounded = False
        pc.on_event("BeginContact", {"GameObjectNameA": "enemy_1", "GameObjectNameB": "ground"})
        self.assertFalse(pc.is_grounded)

    def test_ground_detection_via_end_contact(self):
        """EndContact between player and ground clears is_grounded."""
        pc.is_grounded = True
        pc.on_event("EndContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertFalse(pc.is_grounded)

    def test_jump_when_grounded(self):
        """Jump applies an upward impulse when grounded."""
        pc.is_grounded = True
        pc.pending_jump = True
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_called_once_with(0, -pc.JUMP_IMPULSE)
        self.assertFalse(pc.pending_jump)

    def test_jump_when_airborne_is_queued(self):
        """Jump when airborne is queued (not applied) until grounded."""
        pc.is_grounded = False
        pc.pending_jump = True
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_not_called()
        self.assertTrue(pc.pending_jump)

    def test_event_attack_requested(self):
        """AttackRequested event sets pending_shoot."""
        self.assertFalse(pc.pending_shoot)
        pc.on_event("AttackRequested", {})
        self.assertTrue(pc.pending_shoot)

    def test_event_jump_requested(self):
        """JumpRequested event sets pending_jump."""
        self.assertFalse(pc.pending_jump)
        pc.on_event("JumpRequested", {})
        self.assertTrue(pc.pending_jump)

    def test_movement_right(self):
        """Movement right updates facing and sets velocity."""
        pc.self.get_intent.return_value = 1.0  # move_x > 0
        pc.update(0.016)
        self.assertEqual(pc.facing_x, 1.0)
        pc.self.set_velocity.assert_called_once()
        call_args = pc.self.set_velocity.call_args[0]
        self.assertEqual(call_args[0], pc.PLAYER_SPEED)

    def test_movement_left(self):
        """Movement left updates facing to -1."""
        pc.self.get_intent.return_value = -1.0  # move_x < 0
        pc.update(0.016)
        self.assertEqual(pc.facing_x, -1.0)
        pc.self.set_velocity.assert_called_once()
        call_args = pc.self.set_velocity.call_args[0]
        self.assertEqual(call_args[0], -pc.PLAYER_SPEED)

    def test_movement_none(self):
        """No movement keeps previous facing."""
        pc.facing_x = -1.0
        pc.self.get_intent.return_value = 0
        pc.update(0.016)
        self.assertEqual(pc.facing_x, -1.0)  # unchanged

    def test_shooting_emits_event(self):
        """Shooting emits SpawnProjectile event."""
        pc.pending_shoot = True
        pc.facing_x = 1.0
        pc.update(0.016)
        pc.engine.emit.assert_called_with(
            "SpawnProjectile",
            {"dir_x": 1.0, "dir_y": 0}
        )
        self.assertFalse(pc.pending_shoot)

    def test_shooting_left_direction(self):
        """Shooting left has dir_x = -1."""
        pc.pending_shoot = True
        pc.self.get_intent.return_value = -1.0  # move left to set facing_x = -1.0
        pc.update(0.016)
        pc.engine.emit.assert_called_with(
            "SpawnProjectile",
            {"dir_x": -1.0, "dir_y": 0}
        )

    def test_full_jump_sequence(self):
        """Simulate: grounded -> jump requested -> impulse applied -> airborne (EndContact) ->
        landing (BeginContact) -> jump allowed again."""
        pc.is_grounded = True
        pc.on_event("JumpRequested", {})
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_called_once_with(0, -pc.JUMP_IMPULSE)

        # Box2D reports the body leaving the ground shortly after the impulse.
        pc.on_event("EndContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertFalse(pc.is_grounded)

        # A second jump request while airborne must not apply another impulse.
        pc.self.apply_linear_impulse_to_center.reset_mock()
        pc.on_event("JumpRequested", {})
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_not_called()

        # Landing is reported once the body touches the ground again.
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertTrue(pc.is_grounded)

        # Now a jump is allowed again.
        pc.on_event("JumpRequested", {})
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_called_once_with(0, -pc.JUMP_IMPULSE)


if __name__ == "__main__":
    unittest.main()
