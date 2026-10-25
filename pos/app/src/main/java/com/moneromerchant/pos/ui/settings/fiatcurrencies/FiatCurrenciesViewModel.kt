package com.moneromerchant.pos.ui.settings.fiatcurrencies

import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.navigation.NavHostController
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import com.moneromerchant.pos.data.repository.DataStoreRepository
import com.moneromerchant.pos.data.repository.ExchangeRateRepository
import com.moneromerchant.pos.shared.DataResult
import com.moneromerchant.pos.ui.Settings
import javax.inject.Inject

@HiltViewModel
class FiatCurrenciesViewModel @Inject constructor(
    private val dataStoreRepository: DataStoreRepository,
    private val exchangeRateRepository: ExchangeRateRepository,
) : ViewModel() {

    private val logTag = "FiatCurrenciesViewModel"

    // Supported currencies come from the backend
    var fiatOptions: List<String> by mutableStateOf(emptyList())

    private var navController: NavHostController? = null

    fun setNavController(navController: NavHostController) {
        this.navController = navController
    }

    fun navigateToMainSettings() {
        navController?.popBackStack()
    }

    var primaryFiatCurrency: String by mutableStateOf("")

    var referenceFiatCurrencies: List<String> by mutableStateOf(emptyList())

    init {
        viewModelScope.launch {
            when (val result = exchangeRateRepository.fetchSupportedFiatCurrencies()) {
                is DataResult.Success -> fiatOptions = result.data
                is DataResult.Failure -> Log.w(logTag, "Failed to fetch supported currencies: ${result.message}")
            }
        }
        viewModelScope.launch {
            dataStoreRepository.getPrimaryFiatCurrency().collect { storedPrimaryFiatCurrency ->
                Log.i(logTag, "primaryFiatCurrency: $storedPrimaryFiatCurrency")
                primaryFiatCurrency = storedPrimaryFiatCurrency
            }
        }
        viewModelScope.launch {
            dataStoreRepository.getReferenceFiatCurrencies().collect { storedReferenceFiatCurrencies ->
                Log.i(logTag, "storedContactInformation: $storedReferenceFiatCurrencies")
                referenceFiatCurrencies = storedReferenceFiatCurrencies
            }
        }
    }

    fun updatePrimaryFiatCurrency(newPrimaryFiatCurrency: String) {
        primaryFiatCurrency = newPrimaryFiatCurrency
        viewModelScope.launch {
            dataStoreRepository.savePrimaryFiatCurrency(newPrimaryFiatCurrency)
        }
    }

    fun addReferenceFiatCurrency(newReferenceFiatCurrency: String) {
        referenceFiatCurrencies = referenceFiatCurrencies + newReferenceFiatCurrency
        viewModelScope.launch {
            dataStoreRepository.saveReferenceFiatCurrencies(referenceFiatCurrencies)
        }
    }

    fun removeReferenceFiatCurrency(index: Int) {
        referenceFiatCurrencies = referenceFiatCurrencies.toMutableList().apply { removeAt(index) }
        viewModelScope.launch {
            dataStoreRepository.saveReferenceFiatCurrencies(referenceFiatCurrencies)
        }
    }

    fun moveReferenceFiatCurrencyUp(index: Int) {
        if (index > 0) {
            referenceFiatCurrencies = referenceFiatCurrencies.toMutableList().apply {
                val temp = this[index]
                this[index] = this[index - 1]
                this[index - 1] = temp
            }
            viewModelScope.launch {
                dataStoreRepository.saveReferenceFiatCurrencies(referenceFiatCurrencies)
            }
        }
    }

    fun moveReferenceFiatCurrencyDown(index: Int) {
        if (index < referenceFiatCurrencies.size - 1) {
            referenceFiatCurrencies = referenceFiatCurrencies.toMutableList().apply {
                val temp = this[index]
                this[index] = this[index + 1]
                this[index + 1] = temp
            }
            viewModelScope.launch {
                dataStoreRepository.saveReferenceFiatCurrencies(referenceFiatCurrencies)
            }
        }
    }
}


