// Checkout page JavaScript
// Note: clientKey must be set before this script runs

async function initializeCheckout() {
    try {
        // Call backend to create session
        const response = await fetch('/api/sessions', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });

        if (!response.ok) {
            throw new Error('Failed to create payment session');
        }

        const sessionData = await response.json();

        // Hide loading message
        document.getElementById('loading-container').style.display = 'none';

        // Initialize Adyen Drop-in
        const configuration = {
            clientKey: window.ADYEN_CLIENT_KEY,
            environment: 'test',
            session: {
                id: sessionData.sessionId,
                sessionData: sessionData.sessionData
            },
            onPaymentCompleted: (result, component) => {
                // Handle the result - with Sessions API we need to manually redirect
                if (result.resultCode === 'Authorised' || result.resultCode === 'Pending') {
                    // Redirect with sessionResult parameter (if available)
                    if (result.sessionResult) {
                        window.location.href = '/order/confirmation?sessionId=' + sessionData.sessionId + '&sessionResult=' + encodeURIComponent(result.sessionResult);
                    } else {
                        // Fallback if sessionResult not in response
                        window.location.href = '/order/confirmation?sessionId=' + sessionData.sessionId;
                    }
                } else if (result.resultCode === 'Refused' || result.resultCode === 'Cancelled' || result.resultCode === 'Error') {
                    showError('Payment was declined: ' + result.resultCode + '. Please try again.');
                }
            },
            onError: (error, component) => {
                console.error('Payment error:', error);
                showError('Payment error: ' + (error.message || 'An unknown error occurred'));
            },
            onAdditionalDetails: (state, component) => {
                // Handle additional details if needed
            },
            paymentMethodsConfiguration: {
                card: {
                    hasHolderName: true,
                    holderNameRequired: true,
                    billingAddressRequired: false
                }
            },
            analytics: {
                enabled: true
            }
        };

        const checkout = await AdyenCheckout(configuration);
        const dropin = checkout.create('dropin');
        dropin.mount('#dropin-container');

    } catch (error) {
        console.error('Error initializing checkout:', error);
        document.getElementById('loading-container').style.display = 'none';
        showError('Failed to initialize checkout: ' + error.message);
    }
}

function showError(message) {
    const errorContainer = document.getElementById('error-container');
    errorContainer.textContent = message;
    errorContainer.style.display = 'block';
}

// Initialize checkout when page loads
document.addEventListener('DOMContentLoaded', initializeCheckout);
