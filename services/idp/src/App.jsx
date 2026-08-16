import React, {ReactElement, Suspense, lazy, useState, useEffect} from 'react';
import PropTypes from 'prop-types';

import {MuiThemeProvider} from '@material-ui/core/styles';
import muiTheme from './theme';

import Spinner from './components/Spinner';
import * as version from './version';
import {OpenCloudContext} from './openCloudContext';

const LazyMain = lazy(() => import(/* webpackChunkName: "identifier-main" */ './Main'));

console.info(`Kopano Identifier build version: ${version.build}`); // eslint-disable-line no-console

// config.json, and theme.json's own relative asset paths (theme.common.logo
// etc., see components/ResponsiveScreen.jsx), are served by the main web app
// at the deployment root, not under this app's own /signin/v1 path -- so
// unlike every other asset reference in this file, they can't be resolved
// via process.env.PUBLIC_URL (which is relative to *this* page). Derive the
// deployment root the same way reducers/common.js derives pathPrefix, by
// reading the server-templated data-path-prefix attribute (itself
// "<root>/signin/v1") and stripping the "/signin/v1" suffix back off. Empty
// root ("") is correct for a root-of-domain deployment.
export const deploymentRoot = (() => {
    const root = document.getElementById('root');
    const pathPrefix = root ? root.getAttribute('data-path-prefix') : null;
    return pathPrefix && pathPrefix !== '__PATH_PREFIX__'
        ? pathPrefix.replace(/\/signin\/v1$/, '')
        : '';
})();

const configJsonUrl = `${deploymentRoot}/config.json`;

const App = ({ bgImg }): ReactElement => {
    const [theme, setTheme] = useState(null);
    const [config, setConfig] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchData = async () => {
            try {
                const configResponse = await fetch(configJsonUrl);
                const configData = await configResponse.json();
                setConfig(configData);

                const themeResponse = await fetch(configData.theme);
                const themeData = await themeResponse.json();
                setTheme(themeData);
            } catch (error) {
                console.error('Error fetching config/theme data:', error);
            } finally {
                setLoading(false);
            }
        };

        fetchData();
    }, []);


    if (loading) {
        return <Spinner/>;
    }


    return (
        <OpenCloudContext.Provider value={{theme, config}}>
            <div
                className={`oc-login-bg ${bgImg ? 'oc-login-bg-image' : ''}`}
                style={{backgroundImage: bgImg ? `url(${bgImg})` : undefined}}
            >
                <MuiThemeProvider theme={muiTheme}>
                    <Suspense fallback={<Spinner/>}>
                        <LazyMain/>
                    </Suspense>
                </MuiThemeProvider>
                {!bgImg &&
                    <img
                        src={`${process.env.PUBLIC_URL}/static/icon-lilac.svg`}
                        className={'oc-login-bg-icon'}
                        alt=''
                    />
                }
            </div>
        </OpenCloudContext.Provider>
    );
}

App.propTypes = {
    bgImg: PropTypes.string
};

export default App;
