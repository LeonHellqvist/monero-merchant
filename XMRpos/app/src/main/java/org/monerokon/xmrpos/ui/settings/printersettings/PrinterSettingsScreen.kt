package org.monerokon.xmrpos.ui.settings.printersettings

import android.Manifest
import android.app.Activity
import android.content.Intent
import android.hardware.usb.UsbDevice
import android.net.Uri
import android.provider.Settings
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideInVertically
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.core.app.ActivityCompat
import androidx.navigation.NavHostController
import org.monerokon.xmrpos.R
import org.monerokon.xmrpos.ui.common.composables.BluetoothConnectPermissionTextProvider
import org.monerokon.xmrpos.ui.common.composables.BluetoothScanPermissionTextProvider
import org.monerokon.xmrpos.ui.common.composables.InputTile
import org.monerokon.xmrpos.ui.common.composables.PermissionDialog
import org.monerokon.xmrpos.ui.common.composables.ProportionalRow
import org.monerokon.xmrpos.ui.common.composables.StyledTopAppBar

@Composable
fun PrinterSettingsScreenRoot(viewModel: PrinterSettingsViewModel, navController: NavHostController) {
    viewModel.setNavController(navController)
    PrinterSettingsScreen(
        onBackClick = viewModel::navigateToMainSettings,
        dismissDialog = viewModel::dismissDialog,
        onPermissionResult = viewModel::onPermissionResult,
        visiblePermissionDialogQueue = viewModel.visiblePermissionDialogQueue,
        printerConnectionType = viewModel.printerConnectionType,
        updatePrinterConnectionType = viewModel::updatePrinterConnectionType,
        printerDpi = viewModel.printerDpi,
        updatePrinterDpi = viewModel::updatePrinterDpi,
        printerWidth = viewModel.printerWidth,
        updatePrinterWidth = viewModel::updatePrinterWidth,
        printerNbrCharactersPerLine = viewModel.printerNbrCharactersPerLine,
        updatePrinterNbrCharactersPerLine = viewModel::updatePrinterNbrCharactersPerLine,
        printerCharsetEncoding = viewModel.printerCharsetEncoding,
        updatePrinterCharsetEncoding = viewModel::updatePrinterCharsetEncoding,
        printerCharsetId = viewModel.printerCharsetId,
        updatePrinterCharsetId = viewModel::updatePrinterCharsetId,
        printerAddress = viewModel.printerAddress,
        updatePrinterAddress = viewModel::updatePrinterAddress,
        printerPort = viewModel.printerPort,
        updatePrinterPort = viewModel::updatePrinterPort,
        requestPermissionPrinterUsb = viewModel::requestPermissionPrinterUsb,
        attachedUsbDevices = viewModel.attachedUsbDevices,
        printTest = viewModel::printTest,
        printingInProgress = viewModel.printingInProgress
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PrinterSettingsScreen(
    onBackClick: () -> Unit,
    dismissDialog: () -> Unit,
    onPermissionResult: (String, Boolean) -> Unit,
    visiblePermissionDialogQueue: List<String>,
    printerConnectionType: String,
    updatePrinterConnectionType: (String) -> Unit,
    printerDpi: String,
    updatePrinterDpi: (String) -> Unit,
    printerWidth: String,
    updatePrinterWidth: (String) -> Unit,
    printerNbrCharactersPerLine: String,
    updatePrinterNbrCharactersPerLine: (String) -> Unit,
    printerCharsetEncoding: String,
    updatePrinterCharsetEncoding: (String) -> Unit,
    printerCharsetId: String,
    updatePrinterCharsetId: (String) -> Unit,
    printerAddress: String,
    updatePrinterAddress: (String) -> Unit,
    printerPort: String,
    updatePrinterPort: (String) -> Unit,
    requestPermissionPrinterUsb: (UsbDevice) -> Unit,
    attachedUsbDevices: List<UsbDevice>,
    printTest: () -> Unit,
    printingInProgress: Boolean
) {
    val bluetoothPermissionsToRequest = arrayOf(
        Manifest.permission.BLUETOOTH,
        Manifest.permission.BLUETOOTH_SCAN,
        Manifest.permission.BLUETOOTH_CONNECT
    )

    val bluetoothPermissionResultLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestMultiplePermissions(),
        onResult = { perms ->
            bluetoothPermissionsToRequest.forEach { permission ->
                onPermissionResult(
                    permission,
                    perms[permission] == true
                )
            }
        }
    )

    val activity = LocalContext.current as Activity
    val scrollState = rememberScrollState()

    Scaffold(
        topBar = {
            StyledTopAppBar(
                text = "Printer settings",
                onBackClick = onBackClick
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .padding(innerPadding)
                .padding(horizontal = 24.dp)
                .fillMaxSize()
                .imePadding()
                .verticalScroll(scrollState)
        ) {
            Text("Printer connection", style = MaterialTheme.typography.titleMedium)
            Spacer(modifier = Modifier.height(16.dp))
            Surface(
                color = MaterialTheme.colorScheme.surface,
                shape = MaterialTheme.shapes.extraLarge,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
            ) {
                ProportionalRow(
                    horizontalGap = 4.dp,
                    modifier = Modifier.padding(6.dp)
                ) {
                    ConnectionChip(
                        onClick = {
                            updatePrinterConnectionType("bluetooth")
                            bluetoothPermissionResultLauncher.launch(bluetoothPermissionsToRequest)
                        },
                        selected = printerConnectionType == "bluetooth",
                        label = "Bluetooth"
                    )
                    ConnectionChip(
                        onClick = { updatePrinterConnectionType("tcp/ip") },
                        selected = printerConnectionType == "tcp/ip",
                        label = "TCP/IP"
                    )
                    ConnectionChip(
                        onClick = { updatePrinterConnectionType("usb") },
                        selected = printerConnectionType == "usb",
                        label = "USB"
                    )
                    ConnectionChip(
                        onClick = { updatePrinterConnectionType("none") },
                        selected = printerConnectionType == "none",
                        label = "None"
                    )
                }
            }
            Box(modifier = Modifier.defaultMinSize(minHeight = 137.dp)) {
                Column {
                    AnimatedVisibility(
                        visible = printerConnectionType == "tcp/ip",
                        enter = fadeIn() + slideInVertically(initialOffsetY = { -it })
                    ) {
                        Column {
                            Spacer(modifier = Modifier.height(10.dp))
                            // TODO: REMOVE THIS WHEN FEATURE IS TESTED
                            Text("Untested feature", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary, textAlign = TextAlign.Left, modifier = Modifier.fillMaxWidth())
                            Spacer(modifier = Modifier.height(16.dp))
                            InputTile.Base(
                                contentLeft = {InputTile.TextInput(
                                    value = printerAddress,
                                    onValueChange = { updatePrinterAddress(it) },
                                    prefix = "IP",
                                )},
                                contentRight = {InputTile.TextInput(
                                    value = printerPort,
                                    onValueChange = { updatePrinterPort(it) },
                                    prefix = "Port",
                                )}
                            )
                        }
                    }
                    AnimatedVisibility(
                        visible = printerConnectionType == "usb",
                        enter = fadeIn() + slideInVertically(initialOffsetY = { -it })
                    ) {
                        Column {
                            Spacer(modifier = Modifier.height(16.dp))
                            // TODO: REMOVE THIS WHEN FEATURE IS TESTED
                            Text("Untested feature", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary, textAlign = TextAlign.Left, modifier = Modifier.fillMaxWidth())
                            Spacer(modifier = Modifier.height(8.dp))
                            if (attachedUsbDevices.isNotEmpty()) {
                                Spacer(modifier = Modifier.height(16.dp))
                                Text("Attached USB devices", style = MaterialTheme.typography.titleLarge)
                                Spacer(modifier = Modifier.height(16.dp))
                                attachedUsbDevices.forEach { device ->
                                    OutlinedCard(
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Column(
                                            modifier = Modifier.padding(24.dp)
                                        ) {
                                            Text("Device name: ${device.deviceName}", style = MaterialTheme.typography.labelSmall)
                                            Spacer(modifier = Modifier.height(8.dp))
                                            Text("Device ID: ${device.deviceId}", style = MaterialTheme.typography.labelSmall)
                                            Spacer(modifier = Modifier.height(8.dp))
                                            Text("Vendor ID: ${device.vendorId}", style = MaterialTheme.typography.labelSmall)
                                            Spacer(modifier = Modifier.height(8.dp))
                                            Text("Product ID: ${device.productId}", style = MaterialTheme.typography.labelSmall)
                                            Spacer(modifier = Modifier.height(8.dp))
                                            FilledTonalButton(
                                                onClick = {
                                                    requestPermissionPrinterUsb(
                                                        device
                                                    )
                                                }
                                            ) {
                                                Text("Request permission")
                                            }
                                        }
                                    }
                                    Spacer(modifier = Modifier.height(16.dp))
                                }
                            } else {
                                Spacer(modifier = Modifier.height(70.dp)) // TODO: This should be implemented in a better way
                                Text("No attached USB devices", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.40f))
                            }
                        }
                    }
                }
            }
            Text("Printer parameters", style = MaterialTheme.typography.titleMedium)
            Spacer(modifier = Modifier.height(16.dp))
            Column {
                InputTile.Text(
                    value = printerDpi,
                    onValueChange = { updatePrinterDpi(it) },
                    label = "Printer DPI",
                    prefix = "DPI",
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(modifier = Modifier.height(4.dp))
                InputTile.Text(
                    value = printerWidth,
                    onValueChange = { updatePrinterWidth(it) },
                    label = "Printer width (mm)",
                    prefix = "mm",
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(modifier = Modifier.height(4.dp))
                InputTile.Text(
                    value = printerNbrCharactersPerLine,
                    onValueChange = { updatePrinterNbrCharactersPerLine(it) },
                    label = "Printer characters per line",
                    prefix = "Characters",
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(modifier = Modifier.height(4.dp))
                InputTile.Text(
                    value = printerCharsetEncoding,
                    onValueChange = { updatePrinterCharsetEncoding(it) },
                    label = "Printer charset encoding",
                    prefix = "Encoding",
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(modifier = Modifier.height(4.dp))
                InputTile.Text(
                    value = printerCharsetId,
                    onValueChange = { updatePrinterCharsetId(it) },
                    label = "Printer charset ID",
                    prefix = "ID",
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Spacer(modifier = Modifier.height(32.dp))
            Button (
                onClick = { printTest() },
                enabled = !printingInProgress,
                modifier = Modifier.align(Alignment.CenterHorizontally)
            ) {
                Text("Print test", style = MaterialTheme.typography.labelSmall)
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
            Spacer(modifier = Modifier.height(57.dp))
            visiblePermissionDialogQueue
                .reversed()
                .forEach { permission ->
                    PermissionDialog(
                        permissionTextProvider = when (permission) {
                            Manifest.permission.BLUETOOTH -> {
                                BluetoothConnectPermissionTextProvider()
                            }
                            Manifest.permission.BLUETOOTH_SCAN -> {
                                BluetoothScanPermissionTextProvider()
                            }
                            Manifest.permission.BLUETOOTH_CONNECT -> {
                                BluetoothConnectPermissionTextProvider()
                            }
                            else -> return@forEach
                        },
                        isPermanentlyDeclined = !ActivityCompat.shouldShowRequestPermissionRationale(
                            activity, permission
                        ),
                        onDismiss = dismissDialog,
                        onOkClick = {
                            dismissDialog()
                            bluetoothPermissionResultLauncher.launch(
                                arrayOf(permission)
                            )
                        },
                        onGoToAppSettingsClick = { activity.openAppSettings() }
                    )
                }
        }
    }
}
@Composable
fun ConnectionChip(
    onClick: () -> Unit,
    selected: Boolean,
    label: String,
    modifier: Modifier = Modifier
) {
    val iconWidth by animateDpAsState(
        targetValue = if (selected) FilterChipDefaults.IconSize + 4.dp else 0.dp,
        animationSpec = tween(durationMillis = 200),
        label = "iconWidthAnimation"
    )

    Surface(
        shape = MaterialTheme.shapes.extraLarge,
        color = if (selected) MaterialTheme.colorScheme.surfaceContainerHigh else MaterialTheme.colorScheme.surfaceVariant,
        onClick = onClick,
        modifier = modifier.fillMaxHeight()
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.Center,
            modifier = Modifier.animateContentSize(
                animationSpec = tween(durationMillis = 200)
            )
        ) {
            Row(modifier = Modifier.width(iconWidth)) {
                Icon(
                    painter = painterResource(R.drawable.check_24px),
                    contentDescription = "Selected",
                    modifier = Modifier.size(FilterChipDefaults.IconSize),
                    tint = MaterialTheme.colorScheme.onPrimaryContainer
                )
                Spacer(modifier = Modifier.width(4.dp))
            }

            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall.copy(
                    color = if (selected) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface
                ),
                maxLines = 1
            )
        }
    }
}

fun Activity.openAppSettings() {
    Intent(
        Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
        Uri.fromParts("package", packageName, null)
    ).also(::startActivity)
}