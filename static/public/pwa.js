if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/static/public/sw.js')
        .then(() => console.log("Service Worker registered"))
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


    let deferredPrompt;

    window.addEventListener('beforeinstallprompt', (e) => {
        e.preventDefault();
        deferredPrompt = e;

        document.getElementById(installButtonId)?.addEventListener('click', () => {
            deferredPrompt.prompt();
            deferredPrompt.userChoice.then((choiceResult) => {
                console.log('User choice:', choiceResult.outcome);
                deferredPrompt = null;
            });
        });
    });
})
