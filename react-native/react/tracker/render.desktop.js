'use strict'
/* @flow */

import React from '../base-react'
import BaseComponent from '../base-component'

import Header from './header-render'
import Action from './action-render'
import Bio from './bio-render'
import Proofs from './proofs-render'

export default class Render extends BaseComponent {
  constructor (props) {
    super(props)
  }

  render () {
    return (
      <div style={{backgroundColor: 'red', display: 'flex', flex: 1, flexDirection: 'column'}}>
        <Header />
        <div style={{backgroundColor: 'green', display: 'flex', flex: 1, flexDirection: 'row', height: 480}}>
          <Bio />
          <Proofs />
        </div>
        <Action />
      </div>
    )
  }
}
