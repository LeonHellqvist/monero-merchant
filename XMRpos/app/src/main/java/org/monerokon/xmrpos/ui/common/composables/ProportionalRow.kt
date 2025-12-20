package org.monerokon.xmrpos.ui.common.composables

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.SubcomposeLayout
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import kotlin.math.roundToInt

@Composable
fun ProportionalRow(
    modifier: Modifier = Modifier,    horizontalGap: Dp = 0.dp,
    content: @Composable () -> Unit
) {
    SubcomposeLayout(modifier = modifier) { constraints ->
        // 1. First Pass: Measure children to find their preferred widths.
        val placeables = subcompose(Unit, content).map {
            it.measure(constraints.copy(minWidth = 0))
        }

        // 2. Calculate the total preferred width and any remaining free space.
        val totalPreferredWidth = placeables.sumOf { it.width }
        // Account for the gaps in the free space calculation
        val totalGap = (horizontalGap.toPx() * (placeables.size - 1)).coerceAtLeast(0f)
        val freeSpace = constraints.maxWidth - totalPreferredWidth - totalGap
        val totalHeight = placeables.maxOfOrNull { it.height } ?: 0

        // 3. Second Pass: Re-measure children, giving them a proportional share of the free space.
        val finalPlaceables = subcompose(2, content).mapIndexed { index, measurable ->
            val placeable = placeables[index]
            val proportion = if (totalPreferredWidth > 0) {
                placeable.width.toFloat() / totalPreferredWidth.toFloat()
            } else {
                0f
            }
            val extraWidth = (freeSpace * proportion).roundToInt()
            // Re-measure with the new, calculated width
            measurable.measure(Constraints.fixed(placeable.width + extraWidth, totalHeight))
        }

        // 4. Layout: Place the children one after another, adding the gap.
        layout(constraints.maxWidth, totalHeight) {
            var currentX = 0
            finalPlaceables.forEach { placeable ->
                placeable.placeRelative(x = currentX, y = 0)
                // Increment X by the width of the placed item AND the gap
                currentX += placeable.width + horizontalGap.roundToPx()
            }
        }
    }
}