package org.monerokon.xmrpos.ui.common.dialogs.error

import org.monerokon.xmrpos.ui.common.dialogs.error.ErrorViewModel

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.size
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavHostController
import org.monerokon.xmrpos.R
import org.monerokon.xmrpos.data.repository.AcceptedTransaction
import org.monerokon.xmrpos.ui.common.composables.CustomAlertDialog

/**
 * Error dialog that monitors errors from the error repository and displays them
 */
@Composable
fun ErrorDialog(
    viewModel: ErrorViewModel = hiltViewModel()
) {
    val errorMsg by viewModel.currentError.collectAsStateWithLifecycle()

    errorMsg?.let { message ->
        CustomAlertDialog(
            onDismissRequest = { viewModel.dismissError() },
            onConfirmation = { viewModel.dismissError() },
            dialogTitle = "Error",
            dialogText = message,
            icon = {
                Icon(painter = painterResource(R.drawable.warning_24px), tint = MaterialTheme.colorScheme.primary, contentDescription = "Error")
            },
            confirmButtonText = "Dismiss"
        )
    }
}