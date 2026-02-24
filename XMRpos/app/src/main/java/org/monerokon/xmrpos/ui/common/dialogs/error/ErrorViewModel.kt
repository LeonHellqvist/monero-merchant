package org.monerokon.xmrpos.ui.common.dialogs.error
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.navigation.NavHostController
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import org.monerokon.xmrpos.data.repository.AcceptedTransaction
import org.monerokon.xmrpos.data.repository.ErrorRepository
import org.monerokon.xmrpos.ui.PaymentEntry
import javax.inject.Inject

@HiltViewModel
class ErrorViewModel @Inject constructor(
    private val errorRepository: ErrorRepository,
) : ViewModel() {

    private var navController: NavHostController? = null

    fun setNavController(navController: NavHostController) {
        this.navController = navController
    }

    // Expose only the first error to the UI (FIFO)
    val currentError: StateFlow<String?> = errorRepository.errors
        .map { it.firstOrNull() }
        .stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5000),
            initialValue = null
        )

    fun dismissError() {
        // Tells the repository to remove the oldest error
        errorRepository.consumeFirstError()
    }

}