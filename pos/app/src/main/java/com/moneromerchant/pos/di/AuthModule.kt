package com.moneromerchant.pos.di

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import io.ktor.client.HttpClient
import kotlinx.serialization.json.Json
import com.moneromerchant.pos.data.repository.AuthRepository
import com.moneromerchant.pos.data.repository.DataStoreRepository
import com.moneromerchant.pos.data.repository.StorageRepository
import javax.inject.Named
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object AuthModule {
    @Provides
    @Singleton
    fun provideAuthRepository(
        jsonSerializer: Json,
        dataStoreRepository: DataStoreRepository,
        storageRepository: StorageRepository,
    ): AuthRepository {
        return AuthRepository(jsonSerializer, dataStoreRepository, storageRepository)
    }
}
