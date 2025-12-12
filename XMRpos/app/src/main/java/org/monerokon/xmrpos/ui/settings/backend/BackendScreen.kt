package org.monerokon.xmrpos.ui.settings.backend

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ElevatedButton
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.navigation.NavHostController
import org.monerokon.xmrpos.ui.common.composables.CustomAlertDialog
import org.monerokon.xmrpos.R
import org.monerokon.xmrpos.data.remote.backend.model.BackendHealthResponse
import org.monerokon.xmrpos.shared.DataResult
import org.monerokon.xmrpos.ui.common.composables.InputTile
import org.monerokon.xmrpos.ui.common.composables.StyledTopAppBar


@Composable
fun BackendScreenRoot(viewModel: BackendViewModel, navController: NavHostController) {
    viewModel.setNavController(navController)
    BackendScreen(
        onBackClick = viewModel::navigateToMainSettings,
        confOptions = viewModel.confOptions,
        instanceUrl = viewModel.instanceUrl,
        requestInterval = viewModel.requestInterval,
        conf = viewModel.conf,
        updateRequestInterval = viewModel::updateRequestInterval,
        updateConf = viewModel::updateConf,
        healthStatus = viewModel.healthStatus,
        fetchBackendHealth = viewModel::fetchBackendHealth,
        resetHealthStatus = viewModel::resetHealthStatus,
        logout = viewModel::logout
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BackendScreen(
    onBackClick: () -> Unit,
    confOptions: List<String>,
    instanceUrl: String,
    requestInterval: String,
    conf: String,
    updateRequestInterval: (String) -> Unit,
    updateConf: (String) -> Unit,
    healthStatus: DataResult<BackendHealthResponse>?,
    fetchBackendHealth: () -> Unit,
    resetHealthStatus: () -> Unit,
    logout: () -> Unit
) {
    Scaffold(
        topBar = {
            StyledTopAppBar(
                text = "Backend",
                onBackClick = onBackClick
            )
        },
    ) { innerPadding ->
        Column (
            verticalArrangement = Arrangement.Top,
            modifier = Modifier
                .padding(innerPadding)
                .padding(horizontal = 24.dp)
        ) {
            InputTile.Text(
                value = requestInterval,
                onValueChange = {updateRequestInterval(it)},
                label = "Request interval",
                prefix = "Seconds",
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.fillMaxWidth())
            Spacer(modifier = Modifier.height(4.dp))
            InputTile.Dropdown(
                value = conf,
                onItemSelected = {updateConf(it)},
                label = "Number of confirmations to mark as paid",
                prefix = "Conf",
                items = confOptions,
            )
            Spacer(modifier = Modifier.height(24.dp))
            Row (
                horizontalArrangement = Arrangement.Center,
                modifier = Modifier.fillMaxWidth()
            ) {
                FilledTonalButton(onClick = {fetchBackendHealth()}) {
                    Text("Check health", style = MaterialTheme.typography.labelSmall)
                }
                Spacer(modifier = Modifier.width(10.dp))
                Button(
                    onClick = {logout()},
                    colors = ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.secondary,
                    ),
                ) {
                    Text("Log out", style = MaterialTheme.typography.labelSmall)
                }
            }
        }
        when {healthStatus != null ->
            CustomAlertDialog(
                onDismissRequest = { resetHealthStatus() },
                onConfirmation = { resetHealthStatus() },
                dialogTitle = "Health status",
                dialogText = if (healthStatus is DataResult.Failure) healthStatus.message else null,
                dialogContent = { if (healthStatus is DataResult.Success) HealthView(healthStatus) else null },
                confirmButtonText = "Ok",
                dismissButtonText = "Go back",
                icon = {modifier -> Icon(painter = painterResource(R.drawable.info_24px), tint = MaterialTheme.colorScheme.secondary, contentDescription = "info", modifier = modifier)}
            )
        }
    }
}

@Composable
fun HealthView(healthStatus: DataResult.Success<BackendHealthResponse>) {
    Column {
        Row(
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Status backend: " + healthStatus.data.status, style = MaterialTheme.typography.labelLarge.copy(color = MaterialTheme.colorScheme.primaryContainer))
            HealthIndicator(status = healthStatus.data.status)
        }
        Row(
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(start = 16.dp)
        ) {
            Text("Status PostgreSQL: " + healthStatus.data.services.postgresql, style = MaterialTheme.typography.labelLarge.copy(color = MaterialTheme.colorScheme.primaryContainer))
            HealthIndicator(ok = healthStatus.data.services.postgresql)
        }
        Row(
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(start = 16.dp)
        ) {
            Text("Status MoneroPay: " + healthStatus.data.services.moneroPay.status, style = MaterialTheme.typography.labelLarge.copy(color = MaterialTheme.colorScheme.primaryContainer))
            HealthIndicator(status = healthStatus.data.services.moneroPay.status)
        }
        Row(
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(start = 32.dp)
        ) {
            Text("Status WalletRPC: " + healthStatus.data.services.moneroPay.services.walletRpc, style = MaterialTheme.typography.labelLarge.copy(color = MaterialTheme.colorScheme.primaryContainer))
            HealthIndicator(ok = healthStatus.data.services.moneroPay.services.walletRpc)
        }
        Row(
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(start = 32.dp)
        ) {
            Text("Status PostgreSQL: " + healthStatus.data.services.moneroPay.services.postgresql, style = MaterialTheme.typography.labelLarge.copy(color = MaterialTheme.colorScheme.primaryContainer))
            HealthIndicator(ok = healthStatus.data.services.moneroPay.services.postgresql)
        }
    }
}

@Composable
fun HealthIndicator(status: Int? = null, ok: Boolean? = null) {
    Box(
        modifier = Modifier
            .size(12.dp)
            .background(
                color = if (status == 200 || ok == true) Color(
                    0xFF4CAF50
                ) else MaterialTheme.colorScheme.error,
                shape = CircleShape
            )
    )
}