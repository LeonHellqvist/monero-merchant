package com.moneromerchant.pos.di

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import com.moneromerchant.pos.data.local.datastore.DataStoreLocalDataSource
import com.moneromerchant.pos.data.remote.backend.BackendRemoteDataSource
import com.moneromerchant.pos.data.repository.ExchangeRateRepository
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object ExchangeRateModule {

    @Provides
    @Singleton
    fun provideExchangeRateRepository(
        backendRemoteDataSource: BackendRemoteDataSource,
        dataStoreLocalDataSource: DataStoreLocalDataSource
    ): ExchangeRateRepository {
        return ExchangeRateRepository(backendRemoteDataSource, dataStoreLocalDataSource)
    }
}
