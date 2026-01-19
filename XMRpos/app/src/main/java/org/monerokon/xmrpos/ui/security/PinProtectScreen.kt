// NavGraph.kt
package org.monerokon.xmrpos.ui.security

import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideIn
import androidx.compose.animation.slideOut
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import kotlinx.serialization.Serializable
import org.monerokon.xmrpos.ui.common.composables.CustomOutlinedTextField

@Composable
fun PinProtectScreenRoot(
    navController: NavHostController = rememberNavController(),
    startDestination: PinProtectScreen = PinProtectScreen,
    protectedScreen: @Composable () -> Unit?,
    pinCode: String,
    onPinEntered: () -> Unit = {
        navController.navigate(ProtectedScreen) {
            popUpTo(PinProtectScreen) { inclusive = true }
        }
    },
) {
    Box(Modifier.windowInsetsPadding(WindowInsets.safeDrawing)) {
        NavHost(
            navController = navController,
            startDestination = startDestination,
            enterTransition = {
                slideIn(initialOffset = { fullSize -> IntOffset(fullSize.width, 0) }, animationSpec = tween(300, easing = FastOutSlowInEasing))
            },
            exitTransition = {
                slideOut(targetOffset = { fullSize -> IntOffset(fullSize.width, 0) }, animationSpec = tween(300, easing = FastOutSlowInEasing))
            },
        ) {
            composable<PinProtectScreen> {
                PinProtectScreen(
                    pinCode = pinCode,
                    onPinEntered = onPinEntered
                )
            }
            composable<ProtectedScreen> {
                protectedScreen()
            }
        }
    }
}


// PinProtectScreen
@Composable
fun PinProtectScreen(
    pinCode: String,
    onPinEntered: () -> Unit,
) {
    var enteredPinCode by remember { mutableStateOf("") }
    Column (
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        CustomOutlinedTextField(
            value = enteredPinCode,
            onValueChange = {enteredPinCode = it},
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            label = "Enter your PIN",
            modifier = Modifier.width(280.dp)
        )
        Button(
            onClick = {
                if (enteredPinCode == pinCode) {
                    onPinEntered()
                }
            }
        ) {
            Text("Submit")
        }
    }
}


@Serializable
object PinProtectScreen

@Serializable
object ProtectedScreen
