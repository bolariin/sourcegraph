import { gql } from '@apollo/client'
import { createMockClient } from '@apollo/client/testing'
import { Meta } from '@storybook/react'
import { noop } from 'lodash'
import React from 'react'

import { TemporarySettingsContext } from '../../../settings/temporary/TemporarySettingsProvider'
import {
    InMemoryMockSettingsBackend,
    TemporarySettingsStorage,
} from '../../../settings/temporary/TemporarySettingsStorage'
import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

import { BetaConfirmationModal, BetaConfirmationModalContent } from './BetaConfirmationModal'

const mockClient = createMockClient(
    { contents: JSON.stringify({}) },
    gql`
        query {
            temporarySettings {
                contents
            }
        }
    `
)

const Story: Meta = {
    title: 'web/insights/BetaConfirmationModal',
    decorators: [
        story => (
            <EnterpriseWebStory>{() => <div className="p-3 container web-content">{story()}</div>}</EnterpriseWebStory>
        ),
    ],
}

export default Story

export const BetaModalUI: React.FunctionComponent = () => {
    const settingsStorage = new TemporarySettingsStorage(mockClient, true)

    settingsStorage.setSettingsBackend(new InMemoryMockSettingsBackend({}))

    return (
        <TemporarySettingsContext.Provider value={settingsStorage}>
            <div>
                <h2>Some content</h2>
                <BetaConfirmationModal />
            </div>
        </TemporarySettingsContext.Provider>
    )
}

export const BetaModalContent: React.FunctionComponent = () => (
    <BetaConfirmationModalContent onAccept={noop} onDismiss={noop} />
)
