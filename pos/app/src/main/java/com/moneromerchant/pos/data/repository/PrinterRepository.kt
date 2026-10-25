package com.moneromerchant.pos.data.repository

import android.util.Log
import kotlinx.coroutines.flow.first
import com.moneromerchant.pos.data.printer.PrinterServiceManager
import com.moneromerchant.pos.ui.PaymentSuccess
import java.math.BigDecimal
import java.math.RoundingMode
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

class PrinterRepository(private val printerServiceManager: PrinterServiceManager,
    private val dataStoreRepository: DataStoreRepository,
    private val storageRepository: StorageRepository,
    private val errorRepository: ErrorRepository
) {

    suspend fun printReceipt(paymentSuccess: PaymentSuccess) {
        try {
            val success = printerServiceManager.updateEscPosPrinter()
            Log.i("PrinterRepository", "updateEscPosPrinter: $success")
            if (!success) {
                return
            }
            val companyLogo = storageRepository.readImage("logo.png")
            if (companyLogo != null) {
                printerServiceManager.printPicture(companyLogo)
                printerServiceManager.printSpacer()
            }
            printerServiceManager.printTextCenter(dataStoreRepository.getCompanyName().first())
            printerServiceManager.printTextCenter(dataStoreRepository.getContactInformation().first())
            printerServiceManager.printSpacer()
            printerServiceManager.printText("Date: ${parseIsoDate(paymentSuccess.timestamp)}")
            printerServiceManager.printText("Time: ${parseIsoTime(paymentSuccess.timestamp)}")
            printerServiceManager.printSpacer()
            printerServiceManager.printTextCenter("PURCHASE")
            printerServiceManager.printText("TXID: ${paymentSuccess.txId}")
            printerServiceManager.printText("XMR: ${BigDecimal.valueOf(paymentSuccess.xmrAmount).stripTrailingZeros().toPlainString()}")
            printerServiceManager.printText("${paymentSuccess.primaryFiatCurrency}: ${BigDecimal.valueOf(paymentSuccess.fiatAmount).setScale(2, RoundingMode.HALF_UP).toPlainString()}")
            printerServiceManager.printText("Exchange rate: ${BigDecimal.valueOf(paymentSuccess.exchangeRate).setScale(2, RoundingMode.HALF_UP).toPlainString()} ${paymentSuccess.primaryFiatCurrency} / XMR")
            printerServiceManager.printSpacer()
            printerServiceManager.printTextCenter(dataStoreRepository.getReceiptFooter().first())
            printerServiceManager.printEnd()
        } catch (e: Exception) {
            errorRepository.showError("Could not print: ${e.message}")
        }
    }

    private fun parseIsoDate(isoDate: String): String {
        val formatter = DateTimeFormatter.ISO_DATE_TIME
        val localDateTime = LocalDateTime.parse(isoDate, formatter)
        val dateFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd")
        return localDateTime.format(dateFormatter)
    }

    private fun parseIsoTime(isoDate: String): String {
        val formatter = DateTimeFormatter.ISO_DATE_TIME
        val localDateTime = LocalDateTime.parse(isoDate, formatter)
        val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss")
        return localDateTime.format(timeFormatter)
    }
}