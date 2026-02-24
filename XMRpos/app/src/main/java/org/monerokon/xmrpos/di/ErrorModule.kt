package org.monerokon.xmrpos.di

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import org.monerokon.xmrpos.data.repository.ErrorRepository
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object ErrorModule {

    @Provides
    @Singleton
    fun provideErrorRepository(): ErrorRepository {
        return ErrorRepository()
    }

}