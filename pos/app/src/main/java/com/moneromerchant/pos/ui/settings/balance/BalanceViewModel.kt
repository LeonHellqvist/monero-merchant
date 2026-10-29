package com.moneromerchant.pos.ui.settings.balance

import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.navigation.NavHostController
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import com.moneromerchant.pos.data.remote.backend.model.BackendBalancePosResponse
import com.moneromerchant.pos.shared.DataResult
import com.moneromerchant.pos.ui.Settings
import com.moneromerchant.pos.data.repository.BackendRepository
import com.moneromerchant.pos.data.repository.ExchangeRateRepository
import javax.inject.Inject

@HiltViewModel
class BalanceViewModel @Inject constructor(
    private val backendRepository: BackendRepository,
    private val exchangeRateRepository: ExchangeRateRepository,
) : ViewModel() {

    private val logTag = "BalanceViewModel"

    private var navController: NavHostController? = null

    fun setNavController(navController: NavHostController) {
        this.navController = navController
    }

    fun navigateToMainSettings() {
        navController?.popBackStack()
    }

    var posBalance: DataResult<BackendBalancePosResponse>? by mutableStateOf(null)

    var primaryFiatCurrency by mutableStateOf("")

    var exchangeRate by mutableDoubleStateOf(0.0)
    var rateSources by mutableStateOf(listOf<String>())
    var ratesStale by mutableStateOf(false)


    init {
        viewModelScope.launch {
            posBalance = backendRepository.fetchPosBalance()
        }
        viewModelScope.launch {
            val primaryFiatCurrencyResponse = exchangeRateRepository.getPrimaryFiatCurrency().first()
            primaryFiatCurrency = primaryFiatCurrencyResponse
            when (val exchangeRateResponse = exchangeRateRepository.fetchPrimaryExchangeRate().first()) {
                is DataResult.Failure -> exchangeRate = 0.0
                is DataResult.Success -> {
                    exchangeRate = exchangeRateResponse.data.rates.values.first()
                    rateSources = exchangeRateResponse.data.sources
                    ratesStale = exchangeRateResponse.data.stale
                }
            }
        }
    }
}
