package org.monerokon.xmrpos.data.repository

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class ErrorRepository @Inject constructor() {

    // Using a List as a Queue: Add to end, remove from start
    private val _errors = MutableStateFlow<List<String>>(emptyList())
    val errors: StateFlow<List<String>> = _errors.asStateFlow()

    /**
     * Adds an error to the end of the queue
     */
    fun showError(content: String) {
        _errors.update { currentErrors ->
            currentErrors + content
        }
    }

    /**
     * Removes the oldest error
     */
    fun consumeFirstError() {
        _errors.update { currentErrors ->
            if (currentErrors.isEmpty()) {
                currentErrors
            } else {
                // Drop the first element (FIFO)
                currentErrors.drop(1)
            }
        }
    }
}