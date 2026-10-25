import React, {ReactElement, Suspense, lazy, useState, useEffect, useMemo} from 'react';
import PropTypes from 'prop-types';

import {MuiThemeProvider} from '@material-ui/core/styles';

import Spinner from './components/Spinner';
import * as version from './version';
import {OpenCloudContext} from './openCloudContext';
import {applyLoginTheme, buildMuiTheme, extractIdpTheme} from './loginTheme';

const LazyMain = lazy(() => import(/* webpackChunkName: "identifier-main" */ './Main'));

console.info(`Kopano Identifier build version: ${version.build}`); // eslint-disable-line no-console

const App = ({ bgImg }): ReactElement => {
    const [theme, setTheme] = useState(null);
    const [config, setConfig] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchData = async () => {
            try {
                const configResponse = await fetch('/config.json');
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

    const idpTheme = useMemo(() => extractIdpTheme(theme), [theme]);

    useEffect(() => {
        if (theme) {
            applyLoginTheme(idpTheme);
        }
    }, [idpTheme]);

    const muiTheme = useMemo(() => buildMuiTheme(idpTheme), [idpTheme]);

    // the theme.json login page background image takes precedence over the
    // statically rendered background URL from the index template
    const background = idpTheme.backgroundImage || bgImg;

    if (loading) {
        return <Spinner/>;
    }


    return (
        <OpenCloudContext.Provider value={{theme, config}}>
            <div
                className={`oc-login-bg ${background ? 'oc-login-bg-image' : ''}`}
                style={{backgroundImage: background ? `url(${background})` : undefined}}
            >
                <MuiThemeProvider theme={muiTheme}>
                    <Suspense fallback={<Spinner/>}>
                        <LazyMain/>
                    </Suspense>
                </MuiThemeProvider>
                {!background &&
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
