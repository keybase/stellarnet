running recipe
recipe finished, closing ledger
ledger closed
--
-- PostgreSQL database dump
--

-- Dumped from database version 9.6.6
-- Dumped by pg_dump version 9.6.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

SET search_path = public, pg_catalog;

DROP INDEX IF EXISTS public.signersaccount;
DROP INDEX IF EXISTS public.sellingissuerindex;
DROP INDEX IF EXISTS public.scpquorumsbyseq;
DROP INDEX IF EXISTS public.scpenvsbyseq;
DROP INDEX IF EXISTS public.priceindex;
DROP INDEX IF EXISTS public.ledgersbyseq;
DROP INDEX IF EXISTS public.histfeebyseq;
DROP INDEX IF EXISTS public.histbyseq;
DROP INDEX IF EXISTS public.buyingissuerindex;
DROP INDEX IF EXISTS public.accountbalances;
ALTER TABLE IF EXISTS ONLY public.txhistory DROP CONSTRAINT IF EXISTS txhistory_pkey;
ALTER TABLE IF EXISTS ONLY public.txfeehistory DROP CONSTRAINT IF EXISTS txfeehistory_pkey;
ALTER TABLE IF EXISTS ONLY public.trustlines DROP CONSTRAINT IF EXISTS trustlines_pkey;
ALTER TABLE IF EXISTS ONLY public.storestate DROP CONSTRAINT IF EXISTS storestate_pkey;
ALTER TABLE IF EXISTS ONLY public.signers DROP CONSTRAINT IF EXISTS signers_pkey;
ALTER TABLE IF EXISTS ONLY public.scpquorums DROP CONSTRAINT IF EXISTS scpquorums_pkey;
ALTER TABLE IF EXISTS ONLY public.pubsub DROP CONSTRAINT IF EXISTS pubsub_pkey;
ALTER TABLE IF EXISTS ONLY public.publishqueue DROP CONSTRAINT IF EXISTS publishqueue_pkey;
ALTER TABLE IF EXISTS ONLY public.peers DROP CONSTRAINT IF EXISTS peers_pkey;
ALTER TABLE IF EXISTS ONLY public.offers DROP CONSTRAINT IF EXISTS offers_pkey;
ALTER TABLE IF EXISTS ONLY public.ledgerheaders DROP CONSTRAINT IF EXISTS ledgerheaders_pkey;
ALTER TABLE IF EXISTS ONLY public.ledgerheaders DROP CONSTRAINT IF EXISTS ledgerheaders_ledgerseq_key;
ALTER TABLE IF EXISTS ONLY public.ban DROP CONSTRAINT IF EXISTS ban_pkey;
ALTER TABLE IF EXISTS ONLY public.accounts DROP CONSTRAINT IF EXISTS accounts_pkey;
ALTER TABLE IF EXISTS ONLY public.accountdata DROP CONSTRAINT IF EXISTS accountdata_pkey;
DROP TABLE IF EXISTS public.txhistory;
DROP TABLE IF EXISTS public.txfeehistory;
DROP TABLE IF EXISTS public.trustlines;
DROP TABLE IF EXISTS public.storestate;
DROP TABLE IF EXISTS public.signers;
DROP TABLE IF EXISTS public.scpquorums;
DROP TABLE IF EXISTS public.scphistory;
DROP TABLE IF EXISTS public.pubsub;
DROP TABLE IF EXISTS public.publishqueue;
DROP TABLE IF EXISTS public.peers;
DROP TABLE IF EXISTS public.offers;
DROP TABLE IF EXISTS public.ledgerheaders;
DROP TABLE IF EXISTS public.ban;
DROP TABLE IF EXISTS public.accounts;
DROP TABLE IF EXISTS public.accountdata;
DROP EXTENSION IF EXISTS plpgsql;
DROP SCHEMA IF EXISTS public;
--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA public;


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: accountdata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE accountdata (
    accountid character varying(56) NOT NULL,
    dataname character varying(64) NOT NULL,
    datavalue character varying(112) NOT NULL,
    lastmodified integer DEFAULT 0 NOT NULL
);


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE accounts (
    accountid character varying(56) NOT NULL,
    balance bigint NOT NULL,
    seqnum bigint NOT NULL,
    numsubentries integer NOT NULL,
    inflationdest character varying(56),
    homedomain character varying(32) NOT NULL,
    thresholds text NOT NULL,
    flags integer NOT NULL,
    lastmodified integer NOT NULL,
    CONSTRAINT accounts_balance_check CHECK ((balance >= 0)),
    CONSTRAINT accounts_numsubentries_check CHECK ((numsubentries >= 0))
);


--
-- Name: ban; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE ban (
    nodeid character(56) NOT NULL
);


--
-- Name: ledgerheaders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE ledgerheaders (
    ledgerhash character(64) NOT NULL,
    prevhash character(64) NOT NULL,
    bucketlisthash character(64) NOT NULL,
    ledgerseq integer,
    closetime bigint NOT NULL,
    data text NOT NULL,
    CONSTRAINT ledgerheaders_closetime_check CHECK ((closetime >= 0)),
    CONSTRAINT ledgerheaders_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Name: offers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE offers (
    sellerid character varying(56) NOT NULL,
    offerid bigint NOT NULL,
    sellingassettype integer NOT NULL,
    sellingassetcode character varying(12),
    sellingissuer character varying(56),
    buyingassettype integer NOT NULL,
    buyingassetcode character varying(12),
    buyingissuer character varying(56),
    amount bigint NOT NULL,
    pricen integer NOT NULL,
    priced integer NOT NULL,
    price double precision NOT NULL,
    flags integer NOT NULL,
    lastmodified integer NOT NULL,
    CONSTRAINT offers_amount_check CHECK ((amount >= 0)),
    CONSTRAINT offers_offerid_check CHECK ((offerid >= 0))
);


--
-- Name: peers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE peers (
    ip character varying(15) NOT NULL,
    port integer DEFAULT 0 NOT NULL,
    nextattempt timestamp without time zone NOT NULL,
    numfailures integer DEFAULT 0 NOT NULL,
    CONSTRAINT peers_numfailures_check CHECK ((numfailures >= 0)),
    CONSTRAINT peers_port_check CHECK (((port > 0) AND (port <= 65535)))
);


--
-- Name: publishqueue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE publishqueue (
    ledger integer NOT NULL,
    state text
);


--
-- Name: pubsub; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE pubsub (
    resid character(32) NOT NULL,
    lastread integer
);


--
-- Name: scphistory; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE scphistory (
    nodeid character(56) NOT NULL,
    ledgerseq integer NOT NULL,
    envelope text NOT NULL,
    CONSTRAINT scphistory_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Name: scpquorums; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE scpquorums (
    qsethash character(64) NOT NULL,
    lastledgerseq integer NOT NULL,
    qset text NOT NULL,
    CONSTRAINT scpquorums_lastledgerseq_check CHECK ((lastledgerseq >= 0))
);


--
-- Name: signers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE signers (
    accountid character varying(56) NOT NULL,
    publickey character varying(56) NOT NULL,
    weight integer NOT NULL
);


--
-- Name: storestate; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE storestate (
    statename character(32) NOT NULL,
    state text
);


--
-- Name: trustlines; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE trustlines (
    accountid character varying(56) NOT NULL,
    assettype integer NOT NULL,
    issuer character varying(56) NOT NULL,
    assetcode character varying(12) NOT NULL,
    tlimit bigint NOT NULL,
    balance bigint NOT NULL,
    flags integer NOT NULL,
    lastmodified integer NOT NULL,
    CONSTRAINT trustlines_balance_check CHECK ((balance >= 0)),
    CONSTRAINT trustlines_tlimit_check CHECK ((tlimit > 0))
);


--
-- Name: txfeehistory; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE txfeehistory (
    txid character(64) NOT NULL,
    ledgerseq integer NOT NULL,
    txindex integer NOT NULL,
    txchanges text NOT NULL,
    CONSTRAINT txfeehistory_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Name: txhistory; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE txhistory (
    txid character(64) NOT NULL,
    ledgerseq integer NOT NULL,
    txindex integer NOT NULL,
    txbody text NOT NULL,
    txresult text NOT NULL,
    txmeta text NOT NULL,
    CONSTRAINT txhistory_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Data for Name: accountdata; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: accounts; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO accounts VALUES ('GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 10000000000, 8589934592, 0, NULL, '', 'AQAAAA==', 0, 2);
INSERT INTO accounts VALUES ('GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 999999979999999800, 2, 0, NULL, '', 'AQAAAA==', 0, 2);
INSERT INTO accounts VALUES ('GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 9999999700, 8589934595, 0, NULL, '', 'AQAAAA==', 0, 5);


--
-- Data for Name: ban; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: ledgerheaders; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO ledgerheaders VALUES ('63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '0000000000000000000000000000000000000000000000000000000000000000', '572a2e32ff248a07b0e70fd1f6d318c1facd20b6cc08c33d5775259868125a16', 1, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('30dfcbf166ab60ea97c40d5e72472aa596b97e3cba1a47e70cb658353b0b55bb', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '69b1ded30c0c7a1fdf5306e16643da099d872904d16d9e447b1195be1e1172f0', 2, 1518565358, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76Zu5H9cHd0qAsZj9Q6Y8yIIc6utkf/CO+Lm/0KbsfrzPUAAAAAWoN37gAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAAP0PjS3Rf7hs8xVIBg2g8xKvqOO/n/g/XwgPR/HB7qe9psd7TDAx6H99TBuFmQ9oJnYcpBNFtnkR7EZW+HhFy8AAAAAIN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('4acd52937899082cdbad857fd93b120c5dce512120cfe8353a80db894139bbad', '30dfcbf166ab60ea97c40d5e72472aa596b97e3cba1a47e70cb658353b0b55bb', '7101cc0e3fe98078467524c004d873084f50213d7d12fe1d435d45fb8ead201b', 3, 1518565359, 'AAAACTDfy/Fmq2Dql8QNXnJHKqWWuX48uhpH5wy2WDU7C1W7yK1PoB5FRUibMqjOcVD2uG+uYq5Dx3NPEQMITmGC72cAAAAAWoN37wAAAAAAAAAALLKPbMojH+RR+TSBDKGB/tufH2mL12ccCHr1Jn27yPBxAcwOP+mAeEZ1JMAE2HMIT1AhPX0S/h1DXUX7jq0gGwAAAAMN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('693fef1fd0033956a78f500dc8dc6826930aa9ef9c5e02a20721c36151b36a70', '4acd52937899082cdbad857fd93b120c5dce512120cfe8353a80db894139bbad', 'e56efd02fe3d054467f4b0779593e8f79f9ed9ce347f6e87955a84055ae7d137', 4, 1518565360, 'AAAACUrNUpN4mQgs262Ff9k7EgxdzlEhIM/oNTqA24lBObutUOGuJHNuUTfww+8n8v27D1xYMI3+2pcyfJPQ754hpnMAAAAAWoN38AAAAAAAAAAALG1UZnCzCamYL0dOv0yVmx9sJIXQO2l+Gr9BTnPshPnlbv0C/j0FRGf0sHeVk+j3n57ZzjR/boeVWoQFWufRNwAAAAQN4Lazp2QAAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('5c320ccb5c493f935dbdec7c36932a80770b30b73c6c4f83ff052c02e0fbdd09', '693fef1fd0033956a78f500dc8dc6826930aa9ef9c5e02a20721c36151b36a70', '79bf8e60d38c7ba0223fe78895d570f25c4901bfa31e84642479e595205d0706', 5, 1518565361, 'AAAACWk/7x/QAzlWp49QDcjcaCaTCqnvnF4Cogchw2FRs2pwMIlKwYZPVVKU4XQ6QduLlG2f00NQMIhO3lR8ebmUeYEAAAAAWoN38QAAAAAAAAAAtqD7ZvGzk5mv/AtZ7AfsBkliGMVUtteUvuL+rVvYC/55v45g04x7oCI/54iV1XDyXEkBv6MehGQkeeWVIF0HBgAAAAUN4Lazp2QAAAAAAAAAAAH0AAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('60204650809e0375458cd63c4474d0d78bbe0768bb51a481a531ddc4b81d09b7', '5c320ccb5c493f935dbdec7c36932a80770b30b73c6c4f83ff052c02e0fbdd09', '2495ad2549b289e0d8a5badf31b0899a9a010369d5a4f733d17b0effa982dd36', 6, 1518565362, 'AAAACVwyDMtcST+TXb3sfDaTKoB3CzC3PGxPg/8FLALg+90Jy6l9WRQySqjBUbWJG5O2DEHGTx5jD08ZKwcMAgE1CrcAAAAAWoN38gAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERkkla0lSbKJ4Nilut8xsImamgEDadWk9zPRew7/qYLdNgAAAAYN4Lazp2QAAAAAAAAAAAH0AAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');


--
-- Data for Name: offers; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: peers; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: publishqueue; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: pubsub; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: scphistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO scphistory VALUES ('GCVCY6RIPKIVYXB3DQJ2ZDZX3LZTLYWKVBYSBVHZMSEIOGYGJJ5RRPP2', 2, 'AAAAAKoseih6kVxcOxwTrI832vM14sqocSDU+WSIhxsGSnsYAAAAAAAAAAIAAAACAAAAAQAAAEi7kf1wd3SoCxmP1DpjzIghzq62R/8I74ub/Qpux+vM9QAAAABag3fuAAAAAgAAAAgAAAABAAAACQAAAAgAAAADAAAnEAAAAAAAAAABnD4UmgZrqGcNCEMVoAXZxYnMGuRcYOs6I+7jsN7MzDgAAABAyr4VmboHqLH62JZcdXYqoiVzOM32IKvQNWiBqxYc3zp5xABHBi2GiVWCOX4hnAd974kfCrpCCekwf/222pT4DA==');
INSERT INTO scphistory VALUES ('GCVCY6RIPKIVYXB3DQJ2ZDZX3LZTLYWKVBYSBVHZMSEIOGYGJJ5RRPP2', 3, 'AAAAAKoseih6kVxcOxwTrI832vM14sqocSDU+WSIhxsGSnsYAAAAAAAAAAMAAAACAAAAAQAAADDIrU+gHkVFSJsyqM5xUPa4b65irkPHc08RAwhOYYLvZwAAAABag3fvAAAAAAAAAAAAAAABnD4UmgZrqGcNCEMVoAXZxYnMGuRcYOs6I+7jsN7MzDgAAABA+C2WyjG12PlxsBIHAjrnS+pq/ZjQUFtn8s4F+tj3i+cZzyONLJNNIsrn4aJSbOGHFNoJgf/XbAAbNaf0KNguAQ==');
INSERT INTO scphistory VALUES ('GCVCY6RIPKIVYXB3DQJ2ZDZX3LZTLYWKVBYSBVHZMSEIOGYGJJ5RRPP2', 4, 'AAAAAKoseih6kVxcOxwTrI832vM14sqocSDU+WSIhxsGSnsYAAAAAAAAAAQAAAACAAAAAQAAADBQ4a4kc25RN/DD7yfy/bsPXFgwjf7alzJ8k9DvniGmcwAAAABag3fwAAAAAAAAAAAAAAABnD4UmgZrqGcNCEMVoAXZxYnMGuRcYOs6I+7jsN7MzDgAAABA5VB5bRyaamk2a59RyftaG8VLxi9KAykL3QzjIHLdXvC9alq0mLdRY0jkPhbWhVZ9XK2paPXg8WihpA41zqjxAg==');
INSERT INTO scphistory VALUES ('GCVCY6RIPKIVYXB3DQJ2ZDZX3LZTLYWKVBYSBVHZMSEIOGYGJJ5RRPP2', 5, 'AAAAAKoseih6kVxcOxwTrI832vM14sqocSDU+WSIhxsGSnsYAAAAAAAAAAUAAAACAAAAAQAAADAwiUrBhk9VUpThdDpB24uUbZ/TQ1AwiE7eVHx5uZR5gQAAAABag3fxAAAAAAAAAAAAAAABnD4UmgZrqGcNCEMVoAXZxYnMGuRcYOs6I+7jsN7MzDgAAABAtAPluPlax5aduSxNnhLNkrU4lCaJdNXL0tAgoCHwpYqv9rAQEGX1G8GJyHwysUFGTrIE1HmeOH2PZT+XBFkuAQ==');
INSERT INTO scphistory VALUES ('GCVCY6RIPKIVYXB3DQJ2ZDZX3LZTLYWKVBYSBVHZMSEIOGYGJJ5RRPP2', 6, 'AAAAAKoseih6kVxcOxwTrI832vM14sqocSDU+WSIhxsGSnsYAAAAAAAAAAYAAAACAAAAAQAAADDLqX1ZFDJKqMFRtYkbk7YMQcZPHmMPTxkrBwwCATUKtwAAAABag3fyAAAAAAAAAAAAAAABnD4UmgZrqGcNCEMVoAXZxYnMGuRcYOs6I+7jsN7MzDgAAABATjt40plCx8dExXzswpFvFnBwNC8Q+0jZB4DSIhjppjwBjVBkhVvODOVScn/5Yu7k5cglfqyBTQ2ny2pWzkUzBw==');


--
-- Data for Name: scpquorums; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO scpquorums VALUES ('9c3e149a066ba8670d084315a005d9c589cc1ae45c60eb3a23eee3b0decccc38', 6, 'AAAAAQAAAAEAAAAAqix6KHqRXFw7HBOsjzfa8zXiyqhxINT5ZIiHGwZKexgAAAAA');


--
-- Data for Name: signers; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: storestate; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO storestate VALUES ('databaseschema                  ', '5');
INSERT INTO storestate VALUES ('networkpassphrase               ', 'Test SDF Network ; September 2015');
INSERT INTO storestate VALUES ('forcescponnextlaunch            ', 'false');
INSERT INTO storestate VALUES ('ledgerupgrades                  ', '{
    "time": 0,
    "version": {
        "has": false
    },
    "fee": {
        "has": false
    },
    "maxtxsize": {
        "has": false
    },
    "reserve": {
        "has": false
    }
}');
INSERT INTO storestate VALUES ('lastclosedledger                ', '60204650809e0375458cd63c4474d0d78bbe0768bb51a481a531ddc4b81d09b7');
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v9.1.0",
    "currentLedger": 6,
    "currentBuckets": [
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "e84353fe97afbb9e1d632a0dd946ff3ac4fce36f2e37e7a3778a86540ceb97df"
        },
        {
            "curr": "51e113d7e9f4d92f79c03400f24ae1c0a91013833aff460ebfc970afa280ba24",
            "next": {
                "state": 1,
                "output": "e84353fe97afbb9e1d632a0dd946ff3ac4fce36f2e37e7a3778a86540ceb97df"
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        }
    ]
}');
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAACqLHooepFcXDscE6yPN9rzNeLKqHEg1PlkiIcbBkp7GAAAAAAAAAAGAAAAA5w+FJoGa6hnDQhDFaAF2cWJzBrkXGDrOiPu47DezMw4AAAAAQAAADDLqX1ZFDJKqMFRtYkbk7YMQcZPHmMPTxkrBwwCATUKtwAAAABag3fyAAAAAAAAAAAAAAABAAAAMMupfVkUMkqowVG1iRuTtgxBxk8eYw9PGSsHDAIBNQq3AAAAAFqDd/IAAAAAAAAAAAAAAEAYWLicxdqyz9exeUsZZv0QDkD9GiviRlKqgwkjapNqj4o2Byn/YIatPY1qhvxLNhy6ky66eesQ3c4KwVwyMZcIAAAAAKoseih6kVxcOxwTrI832vM14sqocSDU+WSIhxsGSnsYAAAAAAAAAAYAAAACAAAAAQAAADDLqX1ZFDJKqMFRtYkbk7YMQcZPHmMPTxkrBwwCATUKtwAAAABag3fyAAAAAAAAAAAAAAABnD4UmgZrqGcNCEMVoAXZxYnMGuRcYOs6I+7jsN7MzDgAAABATjt40plCx8dExXzswpFvFnBwNC8Q+0jZB4DSIhjppjwBjVBkhVvODOVScn/5Yu7k5cglfqyBTQ2ny2pWzkUzBwAAAAFcMgzLXEk/k1297Hw2kyqAdwswtzxsT4P/BSwC4PvdCQAAAAAAAAABAAAAAQAAAAEAAAAAqix6KHqRXFw7HBOsjzfa8zXiyqhxINT5ZIiHGwZKexgAAAAA');


--
-- Data for Name: trustlines; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: txfeehistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txfeehistory VALUES ('db398eb4ae89756325643cad21c94e13bfc074b323ee83e141bf701a5d904f1b', 2, 1, 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('f97caffab8c16023a37884165cb0b3ff1aa2daf4000fef49d21efc847ddbfbea', 2, 2, 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('bd486dbdd02d460817671c4a5a7e9d6e865ca29cb41e62d7aaf70a2fee5b36de', 3, 1, 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('857fd8ff2fdf3d24a6781c955d902ae100dc2715814f25e7282b2e567479c65a', 4, 1, 'AAAAAgAAAAMAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('31f210d64fced45a34ea630f944433de5f339b3480c7d3fff92f2b3426fbc594', 5, 1, 'AAAAAgAAAAMAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAFAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+LUAAAAAgAAAAMAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');


--
-- Data for Name: txhistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txhistory VALUES ('db398eb4ae89756325643cad21c94e13bfc074b323ee83e141bf701a5d904f1b', 2, 1, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvkAAAAAAAAAAABVvwF9wAAAEAYjQcPT2G5hqnBmgGGeg9J8l4c1EnUlxklElH9sqZr0971F6OLWfe/m4kpFtI+sI0i1qLit5A0JyWnbhYLW5oD', '2zmOtK6JdWMlZDytIclOE7/AdLMj7oPhQb9wGl2QTxsAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2sVNYGzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('f97caffab8c16023a37884165cb0b3ff1aa2daf4000fef49d21efc847ddbfbea', 2, 2, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBmKpSgvrwKO20XCOfYfXsGEEUtwYaaEfqSu6ymJmlDma+IX6I7IggbUZMocQdZ94IMAfKdQANqXbIO7ysweeMC', '+Xyv+rjBYCOjeIQWXLCz/xqi2vQAD+9J0h78hH3b++oAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rv9MNzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('bd486dbdd02d460817671c4a5a7e9d6e865ca29cb41e62d7aaf70a2fee5b36de', 3, 1, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQB9kmKW2q3v7Qfy8PMekEb1TTI5ixqkI0BogXrOt7gO162Qbkh2dSTUfeDovc0PAafhDXxthVAlsLujlBmyjBAY=', 'vUhtvdAtRggXZxxKWn6dboZcopy0HmLXqvcKL+5bNt4AAAAAAAAAZAAAAAAAAAABAAAAAAAAAAYAAAAAAAAAAA==', 'AAAAAAAAAAEAAAADAAAAAAAAAAMAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAwAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('857fd8ff2fdf3d24a6781c955d902ae100dc2715814f25e7282b2e567479c65a', 4, 1, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAlQL5AAAAAAAAAAAAGu5L5MAAAAQFXNYQ1nN8EmRkvMcd2feWgcArIxRflch7zupcTFCkz8rBJFwobg4z+hs4Umn3TLWL5Nlr1Sf9CP8JcgGXsDSgQ=', 'hX/Y/y/fPSSmeByVXZAq4QDcJxWBTyXnKCsuVnR5xloAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAYAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAwAAAAMAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAAAAAAlQL5AAAAAAAQAAAAAAAAAA');
INSERT INTO txhistory VALUES ('31f210d64fced45a34ea630f944433de5f339b3480c7d3fff92f2b3426fbc594', 5, 1, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAAAAAAAAAAAAAAAAAGu5L5MAAAAQGwsqMIzwrRIIdYGXSzgym6DUbMHrUre+JV1IWsNriIomdY3zGufZtCCV59ADJ9y165PVEFVsX457e4j0M/RWw0=', 'MfIQ1k/O1Fo06mMPlEQz3l8zmzSAx9P/+S8rNCb7xZQAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAYAAAAAAAAAAA==', 'AAAAAAAAAAEAAAAEAAAAAwAAAAUAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAAAAAAlQL5AAAAAAAQAAAAAAAAAAAAAAAgAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7w==');


--
-- Name: accountdata accountdata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accountdata
    ADD CONSTRAINT accountdata_pkey PRIMARY KEY (accountid, dataname);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (accountid);


--
-- Name: ban ban_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ban
    ADD CONSTRAINT ban_pkey PRIMARY KEY (nodeid);


--
-- Name: ledgerheaders ledgerheaders_ledgerseq_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ledgerheaders
    ADD CONSTRAINT ledgerheaders_ledgerseq_key UNIQUE (ledgerseq);


--
-- Name: ledgerheaders ledgerheaders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ledgerheaders
    ADD CONSTRAINT ledgerheaders_pkey PRIMARY KEY (ledgerhash);


--
-- Name: offers offers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY offers
    ADD CONSTRAINT offers_pkey PRIMARY KEY (offerid);


--
-- Name: peers peers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY peers
    ADD CONSTRAINT peers_pkey PRIMARY KEY (ip, port);


--
-- Name: publishqueue publishqueue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY publishqueue
    ADD CONSTRAINT publishqueue_pkey PRIMARY KEY (ledger);


--
-- Name: pubsub pubsub_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY pubsub
    ADD CONSTRAINT pubsub_pkey PRIMARY KEY (resid);


--
-- Name: scpquorums scpquorums_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY scpquorums
    ADD CONSTRAINT scpquorums_pkey PRIMARY KEY (qsethash);


--
-- Name: signers signers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY signers
    ADD CONSTRAINT signers_pkey PRIMARY KEY (accountid, publickey);


--
-- Name: storestate storestate_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY storestate
    ADD CONSTRAINT storestate_pkey PRIMARY KEY (statename);


--
-- Name: trustlines trustlines_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY trustlines
    ADD CONSTRAINT trustlines_pkey PRIMARY KEY (accountid, issuer, assetcode);


--
-- Name: txfeehistory txfeehistory_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txfeehistory
    ADD CONSTRAINT txfeehistory_pkey PRIMARY KEY (ledgerseq, txindex);


--
-- Name: txhistory txhistory_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txhistory
    ADD CONSTRAINT txhistory_pkey PRIMARY KEY (ledgerseq, txindex);


--
-- Name: accountbalances; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX accountbalances ON accounts USING btree (balance) WHERE (balance >= 1000000000);


--
-- Name: buyingissuerindex; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX buyingissuerindex ON offers USING btree (buyingissuer);


--
-- Name: histbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX histbyseq ON txhistory USING btree (ledgerseq);


--
-- Name: histfeebyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX histfeebyseq ON txfeehistory USING btree (ledgerseq);


--
-- Name: ledgersbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ledgersbyseq ON ledgerheaders USING btree (ledgerseq);


--
-- Name: priceindex; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX priceindex ON offers USING btree (price);


--
-- Name: scpenvsbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scpenvsbyseq ON scphistory USING btree (ledgerseq);


--
-- Name: scpquorumsbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scpquorumsbyseq ON scpquorums USING btree (lastledgerseq);


--
-- Name: sellingissuerindex; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sellingissuerindex ON offers USING btree (sellingissuer);


--
-- Name: signersaccount; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX signersaccount ON signers USING btree (accountid);


--
-- PostgreSQL database dump complete
--

