package org.monerokon.xmrpos.ui.common.composables

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuAnchorType
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextDirection
import androidx.compose.ui.unit.dp
import org.monerokon.xmrpos.R

object InputTile {

    /**
     * A private base layout for all input tiles, providing a consistent row structure.
     *
     * This composable is not meant to be used directly. It establishes a common layout with a
     * label on the left and a slot for an input control on the right, both taking up equal space.
     *
     * @param label The text to be displayed in the left-hand side of the tile.
     * @param modifier The [Modifier] to be applied to the entire tile.
     * @param inputContent The composable content for the input control, displayed on the right.
     */
    @Composable
    private fun Base(
        label: String,
        modifier: Modifier = Modifier,
        inputContent: @Composable () -> Unit
    ) {
        Surface(
            modifier = modifier
                .fillMaxWidth(),
            color = MaterialTheme.colorScheme.surface,
            shape = MaterialTheme.shapes.medium
        ) {
            Row(
                modifier = Modifier.padding(start = 16.dp, top = 10.dp, end = 10.dp, bottom = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                Box(modifier = Modifier.weight(1f)) {
                    Text(text = label, style = MaterialTheme.typography.labelSmall)
                }
                Box(modifier = Modifier.weight(1f)) {
                        inputContent()
                }
            }
        }
    }

    /**
     * An [InputTile] that contains a text field for user input.
     *
     * This composable displays a label on the left and a styled [BasicTextField] on the right,
     * suitable for free-form text entry.
     *
     * @param value The current text value to be displayed in the text field.
     * @param onValueChange The callback that is triggered when the input service updates the text.
     * @param label The text to be displayed in the left-hand side of the tile.
     * @param modifier The [Modifier] to be applied to the tile.
     * @param prefix An optional string to be displayed as a prefix inside the text field.
     * @param visualTransformation An optional [VisualTransformation] to apply to the input,
     *   such as for password masking.
     * @param keyboardOptions Software keyboard options that contain configuration like
     *   [androidx.compose.ui.text.input.ImeAction] and [androidx.compose.ui.text.input.KeyboardType].
     * @param enabled Controls the enabled state of the text field. When `false`, it will not be
     *   interactive.
     */
    @Composable
    fun Text(
        value: String,
        onValueChange: (String) -> Unit,
        label: String,
        modifier: Modifier = Modifier,
        prefix: String?,
        visualTransformation: VisualTransformation = VisualTransformation.None,
        keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
        enabled: Boolean = true,
    ) {
        Base(
            label = label,
            modifier = modifier
        ) {
            BasicTextField(
                enabled = enabled,
                value = value,
                onValueChange = onValueChange,
                modifier = modifier
                    // --- FIX IS HERE ---
                    // Add a background color and shape.
                    // It's important that this shape matches the border shape if you have one.
                    .background(
                        color = MaterialTheme.colorScheme.surfaceVariant,
                        shape = MaterialTheme.shapes.small
                    )
                    .height(37.dp),
                textStyle = MaterialTheme.typography.labelSmall.copy(
                    color = MaterialTheme.colorScheme.onBackground,
                    textDirection = TextDirection.Rtl,
                ),
                visualTransformation = visualTransformation,
                keyboardOptions = keyboardOptions,
                singleLine = true,
                decorationBox = { innerTextField ->
                    Row(
                        // Add horizontal padding inside the background
                        modifier = Modifier.padding(all = 10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Box(modifier = Modifier.weight(1f)){
                            if (prefix != null) {
                                Text(
                                    text = prefix,
                                    style = MaterialTheme.typography.labelSmall.copy(
                                        color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.40f)
                                    ),
                                    modifier = Modifier.padding(end = 4.dp) // Space between prefix and text
                                )
                            } // Give the input area a weight so it fills the remaining space
                        }
                        innerTextField()
                    }
                }
            )
        }
    }

    /**
     * An [InputTile] that contains a dropdown menu for selecting from a list of items.
     *
     * This composable displays a label on the left and an [ExposedDropdownMenuBox] on the right,
     * allowing users to select an option from a predefined list.
     *
     * @param value The currently selected item value to be displayed.
     * @param items The list of string items to display in the dropdown menu.
     * @param onItemSelected The callback that is triggered when an item is selected from the menu.
     * @param label The text to be displayed in the left-hand side of the tile.
     * @param modifier The [Modifier] to be applied to the tile.
     * @param prefix An optional string to be displayed as a prefix above the selected value
     *   inside the dropdown box.
     */
    @OptIn(ExperimentalMaterial3Api::class)
    @Composable
    fun Dropdown(
        value: String,
        items: List<String>,
        onItemSelected: (String) -> Unit,
        label: String,
        modifier: Modifier = Modifier,
        prefix: String?,
    ) {
        Base(
            label = label,
            modifier = modifier
        ) {
            var expanded by remember { mutableStateOf(false) }

            ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = {expanded = !expanded}, modifier = modifier) {
                Surface(
                    modifier = Modifier.menuAnchor(type = ExposedDropdownMenuAnchorType.PrimaryNotEditable, enabled = true).fillMaxWidth(),
                    color = MaterialTheme.colorScheme.surfaceVariant,
                    shape = MaterialTheme.shapes.medium,
                    content = {
                        Box(modifier = Modifier.padding(10.dp)) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.SpaceBetween,
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Column {
                                    if (prefix != null) {
                                        Text(prefix, style = MaterialTheme.typography.labelSmall.copy(color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.40f)))
                                        Spacer(modifier = Modifier.height(4.dp))
                                        Text(value, style = MaterialTheme.typography.labelSmall.copy(color = MaterialTheme.colorScheme.onBackground))
                                    }
                                }
                                if (expanded) {
                                    Icon(painter = painterResource(R.drawable.keyboard_arrow_up_24px), tint = MaterialTheme.colorScheme.onBackground, contentDescription = "Arrow up")
                                } else {
                                    Icon(painter = painterResource(R.drawable.keyboard_arrow_down_24px), tint = MaterialTheme.colorScheme.onBackground, contentDescription = "Arrow down")
                                }
                            }
                        }
                    }
                )
                ExposedDropdownMenu(
                    expanded = expanded,
                    onDismissRequest = {expanded = false},
                    containerColor = MaterialTheme.colorScheme.surfaceVariant,
                    shape = MaterialTheme.shapes.small
                    ) {
                    items.forEach { item ->
                        DropdownMenuItem(
                            text = { Text(item, style = MaterialTheme.typography.labelSmall) },
                            onClick = {
                                expanded = false
                                onItemSelected(item)
                            },
                            contentPadding = ExposedDropdownMenuDefaults.ItemContentPadding,
                        )
                    }
                }
            }
        }
    }
}
