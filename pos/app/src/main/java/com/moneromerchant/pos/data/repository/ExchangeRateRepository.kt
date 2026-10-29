package com.moneromerchant.pos.data.repository

import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.flowOn
import com.moneromerchant.pos.data.local.datastore.DataStoreLocalDataSource
import com.moneromerchant.pos.data.remote.backend.BackendRemoteDataSource
import com.moneromerchant.pos.data.remote.exchangeRate.model.ExchangeRateResponse
import com.moneromerchant.pos.shared.DataResult
import javax.inject.Inject

// The backend caches upstream prices itself, so this only saves requests to our own backend
private const val CACHE_DURATION_MS = 60_000L

class ExchangeRateRepository @Inject constructor(
    private val backendRemoteDataSource: BackendRemoteDataSource,
    private val dataStoreLocalDataSource: DataStoreLocalDataSource
) {

    private val logTag = "ExchangeRateRepository"

    @Volatile private var cached: ExchangeRateResponse? = null
    @Volatile private var cachedAt = 0L

    fun fetchPrimaryExchangeRate(): Flow<DataResult<ExchangeRateResponse>> = flow {
        val primaryFiatCurrency = dataStoreLocalDataSource.getPrimaryFiatCurrency().first()
        if (primaryFiatCurrency.isBlank()) {
            emit(DataResult.Success(ExchangeRateResponse(emptyMap(), emptyList(), false)))
            return@flow
        }

        emit(ratesFor(listOf(primaryFiatCurrency)))
    }.flowOn(Dispatchers.IO)

    fun fetchReferenceExchangeRates(): Flow<DataResult<ExchangeRateResponse>> = flow {
        val referenceFiatCurrencies = dataStoreLocalDataSource.getReferenceFiatCurrencies().first().distinct()
        if (referenceFiatCurrencies.isEmpty()) {
            emit(DataResult.Success(ExchangeRateResponse(emptyMap(), emptyList(), false)))
            return@flow
        }

        emit(ratesFor(referenceFiatCurrencies))
    }.flowOn(Dispatchers.IO)

    fun fetchExchangeRates(): Flow<DataResult<ExchangeRateResponse>> = flow {
        val fiatCurrencies = dataStoreLocalDataSource.getFiatCurrencies().first().distinct()
        if (fiatCurrencies.isEmpty()) {
            emit(DataResult.Success(ExchangeRateResponse(emptyMap(), emptyList(), false)))
            return@flow
        }

        emit(ratesFor(fiatCurrencies))
    }.flowOn(Dispatchers.IO)

    fun getPrimaryFiatCurrency(): Flow<String> {
        return dataStoreLocalDataSource.getPrimaryFiatCurrency()
    }

    fun getReferenceFiatCurrencies(): Flow<List<String>> {
        return dataStoreLocalDataSource.getReferenceFiatCurrencies()
    }

    suspend fun fetchSupportedFiatCurrencies(): DataResult<List<String>> {
        return when (val response = backendRemoteDataSource.fetchSupportedCurrencies()) {
            is DataResult.Success -> DataResult.Success(response.data.currencies)
            is DataResult.Failure -> response
        }
    }

    private suspend fun ratesFor(fiatCurrencies: List<String>): DataResult<ExchangeRateResponse> {
        val now = System.currentTimeMillis()
        val cache = cached
        if (cache != null && fiatCurrencies.all { it in cache.rates } && now - cachedAt in 0 until CACHE_DURATION_MS) {
            return DataResult.Success(cache.copy(rates = cache.rates.filterKeys { it in fiatCurrencies }))
        }

        // Fetch every selected currency at once so primary and reference rates share one cache
        val selected = (dataStoreLocalDataSource.getFiatCurrencies().first() + fiatCurrencies)
            .filter { it.isNotBlank() }
            .distinct()

        return when (val response = backendRemoteDataSource.fetchRates(selected)) {
            is DataResult.Success -> {
                val rates = response.data.rates.mapValues { it.value.rate }
                cached = ExchangeRateResponse(rates, response.data.sources, response.data.stale)
                cachedAt = now
                DataResult.Success(ExchangeRateResponse(rates.filterKeys { it in fiatCurrencies }, response.data.sources, response.data.stale))
            }
            is DataResult.Failure -> {
                Log.w(logTag, "Failed to refresh exchange rates: ${response.message}")
                response
            }
        }
    }
}
