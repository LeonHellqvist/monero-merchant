package com.moneromerchant.pos.data.remote.exchangeRate

import retrofit2.http.GET
import retrofit2.http.Query

interface ExchangeRateApi {
    @GET("simple/price")
    suspend fun fetchExchangeRates(
        @Query("ids") ids: String,
        @Query("vs_currencies") vsCurrencies: String
    ): Map<String, Map<String, Double>>
}
