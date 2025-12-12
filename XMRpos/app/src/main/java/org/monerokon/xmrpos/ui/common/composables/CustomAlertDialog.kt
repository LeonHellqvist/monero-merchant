package org.monerokon.xmrpos.ui.common.composables

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialogDefaults
import androidx.compose.material3.BasicAlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties

/**
 * A custom AlertDialog with controlled padding that does not fill the screen width.
 *
 * @param onDismissRequest Called when the user requests to dismiss the dialog, such as by
 *   tapping the scrim or pressing the back button.
 * @param onConfirmation Called when the user clicks the confirmation button.
 * @param dialogTitle The title of the dialog.
 * @param dialogText The main body text of the dialog.
 * @param confirmButtonText The text for the confirmation button. If null, the button is not shown.
 * @param dismissButtonText The text for the dismiss button. If null, the button is not shown.
 * @param icon The optional icon to be displayed at the top of the dialog.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CustomAlertDialog(
    onDismissRequest: (() -> Unit)? = null,
    onConfirmation: (() -> Unit)? = null,
    dialogTitle: String,
    dialogText: String? = null,
    dialogContent: @Composable (() -> Unit)? = null,
    confirmButtonText: String? = null,
    dismissButtonText: String? = null,
    icon: @Composable (Modifier) -> Unit,
) {
    // 1. Use the base `Dialog` composable. It provides the scrim and dismissal handling.
    Dialog(
        onDismissRequest = onDismissRequest ?: {},
        properties = DialogProperties(
            usePlatformDefaultWidth = false,
        ),
    ) {
        Surface(
            color = MaterialTheme.colorScheme.surfaceContainerHigh,
            shape = AlertDialogDefaults.shape,
            modifier = Modifier.fillMaxWidth().padding(24.dp)
        ) {
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier.padding(40.dp)
            ) {
                // This Box is still useful for spacing and alignment
                Box(contentAlignment = Alignment.Center) {
                    // 2. Pass Modifier.size() to the icon lambda
                    icon(Modifier.size(32.dp))
                }

                Spacer(modifier = Modifier.height(14.dp))

                Text(dialogTitle, style = MaterialTheme.typography.titleLarge.copy(color = MaterialTheme.colorScheme.primaryContainer))
                Spacer(modifier = Modifier.height(10.dp))

                if (dialogText != null) {
                    Text(dialogText, textAlign = TextAlign.Center, style = MaterialTheme.typography.labelMedium.copy(color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.4f)))
                }

                if (dialogContent != null) {
                    dialogContent()
                }

                Spacer(modifier = Modifier.height(24.dp))
                Row {
                    if (dismissButtonText != null) {
                        Button(onClick = onDismissRequest ?: {}, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFFEEEEEE))) {
                            Text(dismissButtonText, color = MaterialTheme.colorScheme.primaryContainer)
                        }
                    }
                    if (dismissButtonText != null && confirmButtonText != null) {
                        Spacer(modifier = Modifier.width(8.dp))
                    }
                    if (confirmButtonText != null) {
                        Button(onClick = onConfirmation ?: {}, colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.secondary)) {
                            Text(confirmButtonText)
                        }
                    }
                }

            }
        }
    }
}
