package com.moneromerchant.pos.data.remote.exchangeRate

import com.moneromerchant.pos.data.remote.exchangeRate.model.ExchangeRateResponse
import com.moneromerchant.pos.shared.DataResult
import javax.inject.Inject

class ExchangeRateRemoteDataSource @Inject constructor(
    private val api: ExchangeRateApi
) {
    suspend fun fetchExchangeRates(fromSymbol: String, toSymbols: List<String>): DataResult<ExchangeRateResponse> {
        return try {
            val ids = when (fromSymbol.uppercase()) {
                "XMR", "MONERO" -> "monero"
                else -> fromSymbol.lowercase()
            }
            val vsCurrencies = toSymbols.joinToString(",") { it.lowercase() }
            val response = api.fetchExchangeRates(ids, vsCurrencies)
            val rates = response[ids]?.mapKeys { it.key.uppercase() } ?: emptyMap()
            DataResult.Success(rates)
        } catch (e: Exception) {
            DataResult.Failure(message = e.message ?: "Unknown error")
        }
    }
}