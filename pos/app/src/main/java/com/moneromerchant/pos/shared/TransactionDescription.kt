package com.moneromerchant.pos.shared

object TransactionDescription {
    const val DEFAULT = "Monero Merchant"
    const val MAX_UTF8_BYTES = 128

    fun normalize(value: String): String {
        require(Charsets.UTF_8.newEncoder().canEncode(value)) {
            "Transaction description must be valid Unicode"
        }
        require(value.none { it <= '\u001f' || it in '\u007f'..'\u009f' || it == '\u2028' || it == '\u2029' }) {
            "Transaction description must be a single line without control characters"
        }
        // Unicode White_Space, matching the backend's unicode.IsSpace.
        val normalized = value.trim { it == ' ' || it == '\u00a0' || it == '\u1680' ||
            it in '\u2000'..'\u200a' || it == '\u202f' || it == '\u205f' || it == '\u3000' }
            .ifEmpty { DEFAULT }
        require(normalized.toByteArray(Charsets.UTF_8).size <= MAX_UTF8_BYTES) {
            "Transaction description must be at most 128 UTF-8 bytes"
        }
        return normalized
    }
}
