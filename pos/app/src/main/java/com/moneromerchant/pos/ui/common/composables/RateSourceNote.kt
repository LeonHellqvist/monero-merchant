package com.moneromerchant.pos.ui.common.composables

import androidx.compose.foundation.layout.Column
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier

// Credit for the rate providers (CoinPaprika's free API requires it) and a hint for stale rates
@Composable
fun RateSourceNote(sources: List<String>, stale: Boolean, modifier: Modifier = Modifier) {
    Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = modifier) {
        if (stale) {
            Text(
                text = "Exchange rate may be out of date",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.error,
            )
        }
        if (sources.isNotEmpty()) {
            Text(
                text = "Prices by ${sources.joinToString(" and ")}",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onPrimary.copy(alpha = 0.4f),
            )
        }
    }
}
