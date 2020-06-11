/*
 * Copyright 2018 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package graphite

const epsilon = 0.0001

/*


  def generateSteps(self, minStep):
    """Generate allowed steps with step >= minStep in increasing order."""

    self.checkFinite(minStep)

    if self.binary:
      base = 2.0
      mantissas = [1.0]
      exponent = math.floor(math.log(minStep, 2) - EPSILON)
    else:
      base = 10.0
      mantissas = [1.0, 2.0, 5.0]
      exponent = math.floor(math.log10(minStep) - EPSILON)

    while True:
      multiplier = base ** exponent
      for mantissa in mantissas:
        value = mantissa * multiplier
        if value >= minStep * (1.0 - EPSILON):
          yield value
      exponent += 1

  def computeSlop(self, step, divisor):
    """Compute the slop that would result from step and divisor.

    Return the slop, or None if this combination can't cover the full
    range. See chooseStep() for the definition of "slop".

    """

    bottom = step * math.floor(self.minValue / float(step) + EPSILON)
    top = bottom + step * divisor

    if top >= self.maxValue - EPSILON * step:
      return max(top - self.maxValue, self.minValue - bottom)
    else:
      return None

  def chooseStep(self, divisors=None, binary=False):
    """Choose a nice, pretty size for the steps between axis labels.

    Our main constraint is that the number of divisions must be taken
    from the divisors list. We pick a number of divisions and a step
    size that minimizes the amount of whitespace ("slop") that would
    need to be included outside of the range [self.minValue,
    self.maxValue] if we were to push out the axis values to the next
    larger multiples of the step size.

    The minimum step that could possibly cover the variance satisfies

        minStep * max(divisors) >= variance

    or

        minStep = variance / max(divisors)

    It's not necessarily possible to cover the variance with a step
    that size, but we know that any smaller step definitely *cannot*
    cover it. So we can start there.

    For a sufficiently large step size, it is definitely possible to
    cover the variance, but at some point the slop will start growing.
    Let's define the slop to be

        slop = max(minValue - bottom, top - maxValue)

    Then for a given, step size, we know that

        slop >= (1/2) * (step * min(divisors) - variance)

    (the factor of 1/2 is for the best-case scenario that the slop is
    distributed equally on the two sides of the range). So suppose we
    already have a choice that yields bestSlop. Then there is no need
    to choose steps so large that the slop is guaranteed to be larger
    than bestSlop. Therefore, the maximum step size that we need to
    consider is

        maxStep = (2 * bestSlop + variance) / min(divisors)

    """

    self.binary = binary
    if divisors is None:
      divisors = [4,5,6]
    else:
      for divisor in divisors:
        self.checkFinite(divisor, 'divisor')
        if divisor < 1:
          raise GraphError('Divisors must be greater than or equal to one')

    if self.minValue == self.maxValue:
      if self.minValue == 0.0:
        self.maxValue = 1.0
      elif self.minValue < 0.0:
        self.minValue *= 1.1
        self.maxValue *= 0.9
      else:
        self.minValue *= 0.9
        self.maxValue *= 1.1

    variance = self.maxValue - self.minValue

    bestSlop = None
    bestStep = None
    for step in self.generateSteps(variance / float(max(divisors))):
      if bestSlop is not None and step * min(divisors) >= 2 * bestSlop + variance:
        break
      for divisor in divisors:
        slop = self.computeSlop(step, divisor)
        if slop is not None and (bestSlop is None or slop < bestSlop):
          bestSlop = slop
          bestStep = step

    self.step = bestStep
*/
