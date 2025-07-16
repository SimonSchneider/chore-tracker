if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/static/public/sw.js')
        .catch(err => console.error("Service Worker registration failed:", err));
}

function refreshIfVisible() {
    if (document.visibilityState === 'visible') {
        window.location.reload();
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const refreshButtonId = 'refresh-button';
    const installButtonId = 'install-button';
    const isIOS = /iphone|ipad|ipod/i.test(navigator.userAgent);
    const isStandalone = window.matchMedia('(display-mode: standalone)').matches;
    if (isStandalone && isIOS) {
        document.addEventListener('visibilitychange', refreshIfVisible);
        document.addEventListener('focus', refreshIfVisible);
    }

    document.getElementById(refreshButtonId)?.addEventListener("click", refreshIfVisible)


    const installButton = document.getElementById("install-button")

    if (installButton) {
        window.addEventListener('beforeinstallprompt', (e) => {
            e.preventDefault();
            installButton.addEventListener('click', () => {
                e.prompt();
                e.userChoice.then((choiceResult) => {
                    console.log('User choice:', choiceResult.outcome);
                });
            });
        });
    }
})
