package com.moneromerchant.pos.data.remote.exchangeRate.model

data class ExchangeRateResponse(
    val rates: Map<String, Double>,
    // Shown as credit next to the prices
    val sources: List<String>,
    val stale: Boolean,
)
