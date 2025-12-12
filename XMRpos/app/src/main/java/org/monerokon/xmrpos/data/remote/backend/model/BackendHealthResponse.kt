package org.monerokon.xmrpos.data.remote.backend.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class BackendHealthResponse(
    val status: Int,
    val services: TopLevelServices
)

@Serializable
data class TopLevelServices(
    val postgresql: Boolean,
    @SerialName("MoneroPay")
    val moneroPay: MoneroPayStatus
)

@Serializable
data class MoneroPayStatus(
    val status: Int,
    val services: MoneroPayServices
)

@Serializable
data class MoneroPayServices(
    @SerialName("walletrpc")
    val walletRpc: Boolean,
    val postgresql: Boolean
)

