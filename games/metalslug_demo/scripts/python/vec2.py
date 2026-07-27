"""Simple 2D vector math library for game scripts."""

import math


class Vector:
    """2D vector (x, y) with basic arithmetic operations."""

    def __init__(self, x=0, y=0):
        self.x = x
        self.y = y

    def __repr__(self):
        return f"Vector({self.x}, {self.y})"

    def __eq__(self, other):
        if not isinstance(other, Vector):
            return False
        return self.x == other.x and self.y == other.y

    def add(self, other):
        """Return self + other."""
        return Vector(self.x + other.x, self.y + other.y)

    def sub(self, other):
        """Return self - other."""
        return Vector(self.x - other.x, self.y - other.y)

    def scale(self, s):
        """Return self scaled by scalar s."""
        return Vector(self.x * s, self.y * s)

    def negate(self):
        """Return -self."""
        return Vector(-self.x, -self.y)

    def dot(self, other):
        """Return dot product of self and other."""
        return self.x * other.x + self.y * other.y

    def length_squared(self):
        """Return squared length (cheaper than length for comparisons)."""
        return self.dot(self)

    def length(self):
        """Return length (magnitude) of self."""
        return math.sqrt(self.length_squared())

    def normalize(self):
        """Return self scaled to length 1, or Vector(0,0) if zero vector."""
        l = self.length()
        if l == 0:
            return Vector(0, 0)
        return self.scale(1 / l)

    def distance(self, other):
        """Return distance between self and other."""
        return self.sub(other).length()

    def lerp(self, other, t):
        """Linear interpolation: t=0 -> self, t=1 -> other."""
        return self.add(other.sub(self).scale(t))

    def rotate(self, angle):
        """Rotate by angle (radians), counter-clockwise."""
        sin, cos = math.sin(angle), math.cos(angle)
        return Vector(
            self.x * cos - self.y * sin,
            self.x * sin + self.y * cos
        )


# Convenience constants
Zero = Vector(0, 0)
Up = Vector(0, -1)
Down = Vector(0, 1)
Left = Vector(-1, 0)
Right = Vector(1, 0)
