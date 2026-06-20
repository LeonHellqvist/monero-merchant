package com.moneromerchant.pos.di

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import com.moneromerchant.pos.data.local.datastore.DataStoreLocalDataSource
import com.moneromerchant.pos.data.remote.exchangeRate.ExchangeRateApi
import com.moneromerchant.pos.data.remote.exchangeRate.ExchangeRateRemoteDataSource
import com.moneromerchant.pos.data.repository.ExchangeRateRepository
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import javax.inject.Named
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object ExchangeRateModule {

    @Provides
    @Named("exchangeRateRetrofit")
    fun provideExchangeRateRetrofit(): Retrofit {
        return Retrofit.Builder()
            .baseUrl("https://api.coingecko.com/api/v3/")
            .addConverterFactory(GsonConverterFactory.create())
            .build()
    }

    @Provides
    fun provideExchangeRateApi(@Named("exchangeRateRetrofit") exchangeRateRetrofit: Retrofit): ExchangeRateApi {
        return exchangeRateRetrofit.create(ExchangeRateApi::class.java)
    }

    @Provides
    @Singleton
    fun provideExchangeRateRepository(
        exchangeRateRemoteDataSource: ExchangeRateRemoteDataSource,
        DataStoreLocalDataSource: DataStoreLocalDataSource
    ): ExchangeRateRepository {
        return ExchangeRateRepository(exchangeRateRemoteDataSource, DataStoreLocalDataSource)
    }
}