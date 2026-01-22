document.addEventListener("DOMContentLoaded", function () {
    const setLogoUrl = (url) => {
        const logoTextSelector = document.querySelector(".kc-logo-text");

        if (!logoTextSelector) {
            return
        }

        const link = document.createElement("a");
        link.href = url;
        link.target = "_blank";

        const parent = logoTextSelector.parentNode;
        parent.insertBefore(link, logoTextSelector);
        link.appendChild(logoTextSelector);
    }

    setLogoUrl('https://opencloud.eu')
});