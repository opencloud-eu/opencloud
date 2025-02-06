import React, {ReactElement, Suspense, lazy, useState, useEffect} from 'react';
import PropTypes from 'prop-types';

import {MuiThemeProvider} from '@material-ui/core/styles';
import {defaultTheme} from 'kpop/es/theme';

import 'kpop/static/css/base.css';
import 'kpop/static/css/scrollbar.css';

import Spinner from './components/Spinner';
import * as version from './version';
import {OpenCloudContext} from './openCloudContext';

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


    if (loading) {
        return <Spinner/>;
    }


    return (
        <OpenCloudContext.Provider value={{theme, config}}>
            <div
                className={`oc-login-bg ${bgImg ? 'oc-login-bg-image' : ''}`}
                style={{backgroundImage: bgImg ? `url(${bgImg})` : undefined}}
            >
                <MuiThemeProvider theme={defaultTheme}>
                    <Suspense fallback={<Spinner/>}>
                        <LazyMain/>
                    </Suspense>
                </MuiThemeProvider>
                {!bgImg &&
                    <img
                        src={`${process.env.PUBLIC_URL}/static/icon-lilac.svg`}
                        className={'oc-login-bg-icon'}
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
