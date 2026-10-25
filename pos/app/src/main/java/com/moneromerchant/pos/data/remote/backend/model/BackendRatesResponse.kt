package com.moneromerchant.pos.data.remote.backend.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class BackendRatesResponse(
    val base: String,
    val source: String,
    @SerialName("xmr_updated_at")
    val xmrUpdatedAt: String,
    val stale: Boolean,
    val rates: Map<String, BackendRate>,
)

@Serializable
data class BackendRate(
    val rate: Double,
    val date: String,
)

@Serializable
data class BackendCurrenciesResponse(
    val currencies: List<String>,
)
