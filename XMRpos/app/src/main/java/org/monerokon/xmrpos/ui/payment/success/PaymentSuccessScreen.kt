// PaymentSuccessScreen.kt
package org.monerokon.xmrpos.ui.payment.success

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.slideInHorizontally
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.navigation.NavHostController
import org.monerokon.xmrpos.ui.PaymentSuccess
import org.monerokon.xmrpos.R

@Composable
fun PaymentSuccessScreenRoot(viewModel: PaymentSuccessViewModel, navController: NavHostController, fiatAmount: Double, primaryFiatCurrency: String, txId: String, xmrAmount: Double, exchangeRate: Double, timestamp: String, showPrintReceipt: Boolean) {
    viewModel.setNavController(navController)
    PaymentSuccessScreen(
        navigateToEntry = viewModel::navigateToEntry,
        printReceipt = viewModel::printReceipt,
        fiatAmount = fiatAmount,
        primaryFiatCurrency = primaryFiatCurrency,
        txId = txId,
        xmrAmount = xmrAmount,
        exchangeRate = exchangeRate,
        timestamp = timestamp,
        showPrintReceipt = showPrintReceipt,
        printingInProgress = viewModel.printingInProgress
    )
}

@Composable
fun PaymentSuccessScreen(
    navigateToEntry: () -> Unit,
    printReceipt: (PaymentSuccess) -> Unit,
    fiatAmount: Double,
    primaryFiatCurrency: String,
    txId: String,
    xmrAmount: Double,
    exchangeRate: Double,
    timestamp: String,
    showPrintReceipt: Boolean,
    printingInProgress: Boolean
) {
    Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
        Column (
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier.padding(innerPadding).fillMaxSize(),
        ) {
            Spacer(modifier = Modifier.height(20.dp))
            Image(
                painterResource(R.drawable.success),
                contentDescription = null,
                modifier = Modifier.size(60.dp)
            )
            Spacer(modifier = Modifier.height(14.dp))
            Text("Payment Successful", style = MaterialTheme.typography.titleMedium)
            Spacer(modifier = Modifier.height(10.dp))
            Text("The payment has been confirmed.\nYou can now start a new order.", style = MaterialTheme.typography.labelSmall, textAlign = TextAlign.Center)
            Spacer(modifier = Modifier.height(23.dp))
            Row {
                Button(
                    onClick = {navigateToEntry()},
                ) {
                    Text("New Order")
                }
                if (showPrintReceipt) {
                    Row {
                        Spacer(modifier = Modifier.height(10.dp))
                        FilledTonalButton(
                            onClick = {printReceipt(PaymentSuccess(
                                fiatAmount = fiatAmount,
                                primaryFiatCurrency = primaryFiatCurrency,
                                txId = txId,
                                xmrAmount = xmrAmount,
                                exchangeRate = exchangeRate,
                                timestamp = timestamp,
                                showPrintReceipt = showPrintReceipt
                            ))}
                        ) {
                            Text("Print Receipt")
                            AnimatedVisibility(
                                visible = printingInProgress,
                                enter = fadeIn() + slideInHorizontally(initialOffsetX = { -it })
                            ) {
                                Row {
                                    Spacer(modifier = Modifier.width(8.dp))
                                    CircularProgressIndicator(
                                        modifier = Modifier.size(24.dp)
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
