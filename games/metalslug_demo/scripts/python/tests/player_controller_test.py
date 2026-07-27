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
        pc.velocity_y = 0
        pc.is_grounded = True
        pc.self.reset_mock()
        pc.engine.reset_mock()
        pc.self.get_intent.return_value = 0  # default: no movement

    def test_initial_state(self):
        """Player starts grounded, not moving."""
        self.assertEqual(pc.velocity_y, 0)
        self.assertTrue(pc.is_grounded)
        self.assertEqual(pc.facing_x, 1.0)

    def test_gravity_applied_when_airborne(self):
        """Gravity increases velocity_y each frame while airborne."""
        pc.is_grounded = False
        pc.velocity_y = 0
        pc.update(0.016)  # ~60 FPS frame
        expected_vy = pc.GRAVITY * 0.016
        self.assertAlmostEqual(pc.velocity_y, expected_vy, places=2)

    def test_velocity_clamped_while_grounded(self):
        """While grounded, velocity_y does not accumulate downward each frame."""
        pc.is_grounded = True
        pc.velocity_y = 0
        for _ in range(10):
            pc.update(0.016)
        self.assertEqual(pc.velocity_y, 0)

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
        """Jump applies impulse when grounded."""
        pc.is_grounded = True
        pc.pending_jump = True
        pc.update(0.016)
        # After the jump impulse, gravity still integrates the same frame.
        expected_vy = -pc.JUMP_FORCE + pc.GRAVITY * 0.016
        self.assertAlmostEqual(pc.velocity_y, expected_vy, places=1)
        self.assertFalse(pc.is_grounded)
        self.assertFalse(pc.pending_jump)

    def test_jump_when_airborne(self):
        """Jump when airborne is queued until grounded."""
        pc.is_grounded = False
        pc.velocity_y = -200
        pc.pending_jump = True
        pc.update(0.016)
        # Velocity should increase (less negative) due to gravity, not reset
        self.assertNotEqual(pc.velocity_y, -pc.JUMP_FORCE)
        # Jump is not applied, but pending_jump stays True until grounded
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

    def test_full_jump_arc(self):
        """Simulate jump arc: grounded -> jump -> airborne -> land (via BeginContact)."""
        pc.is_grounded = True
        pc.pending_jump = True

        # Frame 1: jump applied
        pc.update(0.016)
        expected_vy = -pc.JUMP_FORCE + pc.GRAVITY * 0.016
        self.assertAlmostEqual(pc.velocity_y, expected_vy, places=1)
        self.assertFalse(pc.is_grounded)

        # Frames 2-35: ascending then descending (peak is ~frame 31-32)
        for _ in range(2, 36):
            pc.update(0.016)

        # By frame 35, should be falling (velocity_y > 0)
        self.assertGreater(pc.velocity_y, 0, f"Should be falling by frame 35, but vy={pc.velocity_y}")

        # Simulate landing: the engine reports contact once the body reaches the ground.
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertTrue(pc.is_grounded, "Should be grounded after impact")

        # Next update should clamp velocity_y back to 0 instead of continuing to fall.
        pc.update(0.016)
        self.assertEqual(pc.velocity_y, 0)


if __name__ == "__main__":
    unittest.main()
