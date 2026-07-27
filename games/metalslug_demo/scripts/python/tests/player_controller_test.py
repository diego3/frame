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
        """Reset global state before each test. Default: resting on one ground contact
        (matches the module's own initial state -- the player spawns on the ground)."""
        pc.facing_x = 1.0
        pc.pending_shoot = False
        pc.pending_jump = False
        pc._ground_contacts = 1
        pc.is_grounded = True
        pc.self.reset_mock()
        pc.engine.reset_mock()
        pc.self.get_intent.return_value = 0  # default: no movement
        pc.self.get_velocity.return_value = 0  # default: at rest vertically

    def _set_airborne(self):
        pc._ground_contacts = 0
        pc.is_grounded = False

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
        self._set_airborne()
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertTrue(pc.is_grounded)

    def test_ground_detection_via_begin_contact_name_order_reversed(self):
        """BeginContact sets is_grounded regardless of which name is A vs B."""
        self._set_airborne()
        pc.on_event("BeginContact", {"GameObjectNameA": "ground", "GameObjectNameB": "player"})
        self.assertTrue(pc.is_grounded)

    def test_ground_detection_via_crate_contact(self):
        """BeginContact against a crate also counts as grounded."""
        self._set_airborne()
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "crate_2"})
        self.assertTrue(pc.is_grounded)

    def test_ground_detection_ignores_unrelated_contact(self):
        """BeginContact between unrelated objects does not affect the player."""
        self._set_airborne()
        pc.on_event("BeginContact", {"GameObjectNameA": "enemy_1", "GameObjectNameB": "ground"})
        self.assertFalse(pc.is_grounded)

    def test_ground_detection_via_end_contact(self):
        """EndContact between player and ground clears is_grounded."""
        pc.on_event("EndContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertFalse(pc.is_grounded)

    def test_grounded_survives_leaving_a_secondary_contact(self):
        """Regression test: touching the ground AND a crate at once, then only separating from the
        crate, must NOT clear is_grounded -- the player is still resting on the ground. A plain
        overwritten boolean (instead of a contact count) got this wrong: the crate's EndContact
        would clobber is_grounded to False even though contact with "ground" was never lost,
        which looked like jumping randomly breaking during normal movement near crates/enemies."""
        # setUp already simulates one contact (the ground). Add a second, incidental one.
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "crate_1"})
        self.assertTrue(pc.is_grounded)

        # Separate from the crate only -- the ground contact never ended.
        pc.on_event("EndContact", {"GameObjectNameA": "player", "GameObjectNameB": "crate_1"})
        self.assertTrue(pc.is_grounded, "should still be grounded via the ground contact")

    def test_ground_contact_count_does_not_go_negative(self):
        """A stray EndContact with no matching BeginContact must not leave the counter negative
        (which would require multiple BeginContacts to ever become grounded again)."""
        self._set_airborne()
        pc.on_event("EndContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertEqual(pc._ground_contacts, 0)
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertTrue(pc.is_grounded)

    def test_jump_when_grounded(self):
        """Jump applies an upward impulse when grounded."""
        pc.pending_jump = True
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_called_once_with(0, -pc.JUMP_IMPULSE)
        self.assertFalse(pc.pending_jump)

    def test_jump_request_dropped_when_airborne(self):
        """JumpRequested while airborne is dropped immediately, not queued for landing --
        queuing made the jump appear to fire on its own whenever the player next touched down."""
        self._set_airborne()
        pc.on_event("JumpRequested", {})
        self.assertFalse(pc.pending_jump)

    def test_update_does_not_apply_impulse_when_not_grounded(self):
        """Defensive check in update(): even if pending_jump were somehow set while airborne, no
        impulse is applied until grounded."""
        self._set_airborne()
        pc.pending_jump = True
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_not_called()
        self.assertTrue(pc.pending_jump)

    def test_event_attack_requested(self):
        """AttackRequested event sets pending_shoot."""
        self.assertFalse(pc.pending_shoot)
        pc.on_event("AttackRequested", {})
        self.assertTrue(pc.pending_shoot)

    def test_event_jump_requested_when_grounded(self):
        """JumpRequested event sets pending_jump when grounded."""
        self.assertFalse(pc.pending_jump)
        pc.on_event("JumpRequested", {})
        self.assertTrue(pc.pending_jump)

    def test_event_jump_requested_ignored_when_airborne(self):
        """JumpRequested event is ignored (not queued) when airborne."""
        self._set_airborne()
        pc.on_event("JumpRequested", {})
        self.assertFalse(pc.pending_jump)

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
        """Simulate: grounded -> jump -> airborne (EndContact) -> jump request dropped (no queue)
        -> landing (BeginContact) -> jump allowed again."""
        pc.on_event("JumpRequested", {})
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_called_once_with(0, -pc.JUMP_IMPULSE)

        # Box2D reports the body leaving the ground shortly after the impulse.
        pc.on_event("EndContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertFalse(pc.is_grounded)

        # A jump request while airborne is dropped, not queued.
        pc.self.apply_linear_impulse_to_center.reset_mock()
        pc.on_event("JumpRequested", {})
        self.assertFalse(pc.pending_jump)
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_not_called()

        # Landing is reported once the body touches the ground again.
        pc.on_event("BeginContact", {"GameObjectNameA": "player", "GameObjectNameB": "ground"})
        self.assertTrue(pc.is_grounded)

        # Now a fresh jump request is allowed again.
        pc.on_event("JumpRequested", {})
        pc.update(0.016)
        pc.self.apply_linear_impulse_to_center.assert_called_once_with(0, -pc.JUMP_IMPULSE)


if __name__ == "__main__":
    unittest.main()
