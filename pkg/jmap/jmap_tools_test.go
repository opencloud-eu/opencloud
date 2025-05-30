package jmap

import (
	"encoding/json"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/require"
)

const jmap_email_with_text_and_html_bodies = `
        {
          "id": "mk2aaadcx",
          "blobId": "cby92nwhy2pswwygvnavdnv0zc3kffdafeauqko2lyw1qvtnjhztwaya72ma",
          "threadId": "dcx",
          "mailboxIds": {
            "a": true
          },
          "keywords": {},
          "size": 9284,
          "receivedAt": "2025-05-30T08:53:32Z",
          "messageId": [
            "1748595212.4933355@example.com"
          ],
          "inReplyTo": null,
          "references": null,
          "sender": null,
          "from": [
            {
              "name": "Superb Openly",
              "email": "superb.openly@example.com"
            }
          ],
          "to": [
            {
              "name": "alan",
              "email": "alan@example.com"
            }
          ],
          "cc": null,
          "bcc": null,
          "replyTo": null,
          "subject": "libero ad blandit rutrum lacinia consectetur sem",
          "sentAt": "2025-05-30T10:53:32+02:00",
          "hasAttachment": false,
          "preview": "Diam egestas imperdiet non eu quam semper euismod netus venenatis ante magnis mus finibus maecenas nec cras ac commodo nascetur aliquet habitasse porta velit felis tempus ligula vulputate. Elit dolor fames neque hac nunc ornare nibh facilisis nisl finib...",
          "bodyValues": {
            "1": {
              "isEncodingProblem": false,
              "isTruncated": false,
              "value": "Diam egestas imperdiet non eu quam semper euismod netus venenatis ante magnis mus finibus maecenas nec cras ac commodo nascetur aliquet habitasse porta velit felis tempus ligula vulputate. Elit dolor fames neque hac nunc ornare nibh facilisis nisl finibus magna senectus montes vulputate justo dis cras interdum. Convallis montes urna iaculis etiam mauris lorem tristique accumsan erat tincidunt posuere quis felis primis dolor a ipsum hendrerit parturient dictum pulvinar phasellus id porttitor. Etiam mi sollicitudin justo eu natoque eros malesuada nostra vulputate maximus habitant arcu fames imperdiet odio at netus eget maecenas elit parturient hendrerit nibh dui augue quisque tellus amet platea sit. Lectus risus potenti bibendum gravida fringilla sollicitudin sit enim consectetur ipsum accumsan parturient lorem varius sagittis rutrum montes vehicula nec mus hendrerit hac malesuada vel ac integer elementum.\nEuismod donec aliquet suspendisse mi blandit faucibus egestas adipiscing purus congue id himenaeos aenean per. Nullam habitasse est volutpat montes laoreet posuere eget suscipit, ultrices interdum mi nulla ac at eleifend praesent dis nostra massa habitant sapien integer porta consequat amet. Ut conubia amet vulputate ridiculus euismod fermentum libero auctor, mus donec eros a netus ad condimentum morbi facilisi neque tellus feugiat class erat metus inceptos.\nAenean himenaeos ridiculus risus dictum taciti dui quisque penatibus interdum magnis sollicitudin commodo tempor ultrices dapibus mi tempus ullamcorper nibh volutpat justo consequat fusce amet hendrerit laoreet dignissim sit venenatis semper libero mus suscipit. Quis aptent non varius porttitor aliquam iaculis justo facilisi nostra sodales imperdiet integer odio tincidunt quisque rhoncus ullamcorper laoreet tristique dolor. Blandit fringilla adipiscing dictumst euismod magnis volutpat tortor mollis varius elementum nostra litora porttitor habitant convallis risus urna consectetur eleifend suspendisse auctor posuere. Senectus mauris purus a dis tincidunt parturient tortor proin facilisis taciti tellus egestas dui ante in turpis adipiscing lacus neque quisque sagittis tristique suscipit est vestibulum nullam. Pellentesque massa ligula lobortis habitasse rutrum finibus fermentum hac egestas augue aliquet efficitur volutpat mattis ac imperdiet malesuada id etiam turpis tempus tellus interdum quisque at pulvinar nullam proin velit dictumst. Habitant rutrum sit dignissim porta luctus aenean volutpat aliquam arcu lacus tincidunt augue mattis porttitor neque congue risus nostra ridiculus dui molestie maximus libero justo faucibus.\nNisl condimentum pulvinar vulputate quam ante urna habitasse suscipit, volutpat lorem venenatis sem dignissim sapien penatibus ipsum felis faucibus eget velit sociosqu dictumst arcu viverra erat. Vitae auctor lobortis etiam ligula maecenas aptent fringilla, tempus pellentesque euismod neque sociosqu posuere curabitur venenatis dis elit inceptos ullamcorper natoque. Suspendisse elementum semper diam luctus odio fringilla sem nascetur blandit nam cubilia integer senectus in sociosqu sollicitudin nisi parturient. Ante maximus donec hac malesuada nisl quam massa nunc justo conubia fringilla tellus natoque scelerisque cubilia litora.\nAliquet morbi ligula quisque dapibus ultrices eros sem malesuada lobortis massa litora vestibulum varius commodo egestas tincidunt aenean ullamcorper at duis velit auctor parturient. Feugiat natoque posuere orci rhoncus ante mus quam, quis non sapien ut purus volutpat himenaeos et senectus fermentum placerat elementum augue. Natoque id mauris vel mus molestie elementum fames hac consectetur sed platea ad eget efficitur maecenas conubia morbi justo vivamus placerat curae pretium nisi ipsum imperdiet. Velit eros volutpat efficitur pharetra natoque primis luctus nunc lacus fusce dolor sagittis porta maecenas odio rutrum dis consectetur nam metus venenatis ut. Iaculis turpis luctus per orci taciti vehicula amet ad integer, quis litora mauris praesent ullamcorper cursus faucibus at eros dictum dolor morbi mus semper senectus laoreet felis torquent. Phasellus senectus nibh ornare dui convallis orci consequat enim justo etiam himenaeos dictum velit dis magna tempor maecenas fermentum luctus morbi molestie praesent condimentum hendrerit penatibus nisl tempus."
            },
            "2": {
              "isEncodingProblem": false,
              "isTruncated": false,
              "value": "<p>Diam egestas imperdiet non eu quam semper euismod netus venenatis ante magnis mus finibus maecenas nec cras ac commodo nascetur aliquet habitasse porta velit felis tempus ligula vulputate. Elit dolor fames neque hac nunc ornare nibh facilisis nisl finibus magna senectus montes vulputate justo dis cras interdum. Convallis montes urna iaculis etiam mauris lorem tristique accumsan erat tincidunt posuere quis felis primis dolor a ipsum hendrerit parturient dictum pulvinar phasellus id porttitor. Etiam mi sollicitudin justo eu natoque eros malesuada nostra vulputate maximus habitant arcu fames imperdiet odio at netus eget maecenas elit parturient hendrerit nibh dui augue quisque tellus amet platea sit. Lectus risus potenti bibendum gravida fringilla sollicitudin sit enim consectetur ipsum accumsan parturient lorem varius sagittis rutrum montes vehicula nec mus hendrerit hac malesuada vel ac integer elementum.</p>\n<p>Euismod donec aliquet suspendisse mi blandit faucibus egestas adipiscing purus congue id himenaeos aenean per. Nullam habitasse est volutpat montes laoreet posuere eget suscipit, ultrices interdum mi nulla ac at eleifend praesent dis nostra massa habitant sapien integer porta consequat amet. Ut conubia amet vulputate ridiculus euismod fermentum libero auctor, mus donec eros a netus ad condimentum morbi facilisi neque tellus feugiat class erat metus inceptos.</p>\n<p>Aenean himenaeos ridiculus risus dictum taciti dui quisque penatibus interdum magnis sollicitudin commodo tempor ultrices dapibus mi tempus ullamcorper nibh volutpat justo consequat fusce amet hendrerit laoreet dignissim sit venenatis semper libero mus suscipit. Quis aptent non varius porttitor aliquam iaculis justo facilisi nostra sodales imperdiet integer odio tincidunt quisque rhoncus ullamcorper laoreet tristique dolor. Blandit fringilla adipiscing dictumst euismod magnis volutpat tortor mollis varius elementum nostra litora porttitor habitant convallis risus urna consectetur eleifend suspendisse auctor posuere. Senectus mauris purus a dis tincidunt parturient tortor proin facilisis taciti tellus egestas dui ante in turpis adipiscing lacus neque quisque sagittis tristique suscipit est vestibulum nullam. Pellentesque massa ligula lobortis habitasse rutrum finibus fermentum hac egestas augue aliquet efficitur volutpat mattis ac imperdiet malesuada id etiam turpis tempus tellus interdum quisque at pulvinar nullam proin velit dictumst. Habitant rutrum sit dignissim porta luctus aenean volutpat aliquam arcu lacus tincidunt augue mattis porttitor neque congue risus nostra ridiculus dui molestie maximus libero justo faucibus.</p>\n<p>Nisl condimentum pulvinar vulputate quam ante urna habitasse suscipit, volutpat lorem venenatis sem dignissim sapien penatibus ipsum felis faucibus eget velit sociosqu dictumst arcu viverra erat. Vitae auctor lobortis etiam ligula maecenas aptent fringilla, tempus pellentesque euismod neque sociosqu posuere curabitur venenatis dis elit inceptos ullamcorper natoque. Suspendisse elementum semper diam luctus odio fringilla sem nascetur blandit nam cubilia integer senectus in sociosqu sollicitudin nisi parturient. Ante maximus donec hac malesuada nisl quam massa nunc justo conubia fringilla tellus natoque scelerisque cubilia litora.</p>\n<p>Aliquet morbi ligula quisque dapibus ultrices eros sem malesuada lobortis massa litora vestibulum varius commodo egestas tincidunt aenean ullamcorper at duis velit auctor parturient. Feugiat natoque posuere orci rhoncus ante mus quam, quis non sapien ut purus volutpat himenaeos et senectus fermentum placerat elementum augue. Natoque id mauris vel mus molestie elementum fames hac consectetur sed platea ad eget efficitur maecenas conubia morbi justo vivamus placerat curae pretium nisi ipsum imperdiet. Velit eros volutpat efficitur pharetra natoque primis luctus nunc lacus fusce dolor sagittis porta maecenas odio rutrum dis consectetur nam metus venenatis ut. Iaculis turpis luctus per orci taciti vehicula amet ad integer, quis litora mauris praesent ullamcorper cursus faucibus at eros dictum dolor morbi mus semper senectus laoreet felis torquent. Phasellus senectus nibh ornare dui convallis orci consequat enim justo etiam himenaeos dictum velit dis magna tempor maecenas fermentum luctus morbi molestie praesent condimentum hendrerit penatibus nisl tempus.</p>"
            }
          },
          "textBody": [
            {
              "partId": "1",
              "blobId": "cfy92nwhy2pswwygvnavdnv0zc3kffdafeauqko2lyw1qvtnjhztwaya72mmga3iee",
              "size": 4328,
              "name": null,
              "type": "text/plain",
              "charset": "utf-8",
              "disposition": null,
              "cid": null,
              "language": null,
              "location": null
            }
          ],
          "htmlBody": [
            {
              "partId": "2",
              "blobId": "cfy92nwhy2pswwygvnavdnv0zc3kffdafeauqko2lyw1qvtnjhztwaya72mimjulei",
              "size": 4363,
              "name": null,
              "type": "text/html",
              "charset": "utf-8",
              "disposition": null,
              "cid": null,
              "language": null,
              "location": null
            }
          ],
          "attachments": []
        }
`

const jmap_email_with_text_body = `
        {
          "id": "mliaaadc7",
          "blobId": "cc0tuhkv1lncttirzg9p97wd7k7gezaz9fbwjir31wrcvkykbm1zkaya9ima",
          "threadId": "dc7",
          "mailboxIds": {
            "a": true
          },
          "keywords": {},
          "size": 4080,
          "receivedAt": "2025-05-30T09:59:55Z",
          "messageId": [
            "1748599195.5902335@example.com"
          ],
          "inReplyTo": null,
          "references": null,
          "sender": null,
          "from": [
            {
              "name": "Cunning Properly",
              "email": "cunning.properly@example.com"
            }
          ],
          "to": [
            {
              "name": "alan",
              "email": "alan@example.com"
            }
          ],
          "cc": null,
          "bcc": null,
          "replyTo": null,
          "subject": "Parturient Nostra Orci",
          "sentAt": "2025-05-30T11:59:55+02:00",
          "hasAttachment": false,
          "preview": "Et magnis pulvinar congue aliquet tincidunt morbi lobortis mattis mus litora malesuada fringilla varius ullamcorper parturient fames accumsan faucibus erat. Magna id est cras eu a netus orci ridiculus lobortis urna dis ipsum at fermentum mi lacinia quis...",
          "bodyValues": {
            "0": {
              "isEncodingProblem": false,
              "isTruncated": false,
              "value": "Et magnis pulvinar congue aliquet tincidunt morbi lobortis mattis mus litora malesuada fringilla varius ullamcorper parturient fames accumsan faucibus erat. Magna id est cras eu a netus orci ridiculus lobortis urna dis ipsum at fermentum mi lacinia quis fames. Cursus ipsum gravida libero ultricies pretium montes rutrum suscipit tempor hac dapibus senectus commodo elementum leo nullam auctor litora pulvinar finibus. Ad nulla torquent quis mollis phasellus sodales aliquet lacinia varius, adipiscing enim habitant et netus egestas eu tempor malesuada mattis hac fusce integer diam inceptos venenatis turpis. Sem senectus aptent non dolor hendrerit magna mauris facilisis justo quam fringilla cursus gravida praesent malesuada taciti odio etiam magnis nostra vivamus. Tempus fames faucibus massa rutrum sit habitant morbi curabitur integer erat et condimentum tincidunt tempor libero vulputate maecenas rhoncus turpis congue a luctus aenean tristique lacinia cursus est fusce non mollis justo euismod facilisis egestas.\nAuctor maecenas vestibulum aenean accumsan eros ex potenti sociosqu, fusce sapien quis faucibus aliquam vivamus tristique hendrerit in per fermentum cras sodales curabitur scelerisque. Finibus metus adipiscing taciti eget rutrum vitae arcu torquent, dignissim at nibh venenatis facilisis molestie erat augue massa feugiat aliquam sollicitudin elementum cursus in. Est neque cras aenean felis justo euismod adipiscing magnis sagittis ut massa aliquet malesuada laoreet velit purus suspendisse bibendum pharetra litora ultrices diam ullamcorper volutpat venenatis egestas. Non laoreet eu interdum sodales phasellus morbi risus maecenas parturient auctor senectus urna ornare faucibus sociosqu habitant nisi cubilia viverra diam fames condimentum tempor scelerisque iaculis lacus elit feugiat adipiscing vivamus. Euismod volutpat gravida fames nascetur ridiculus iaculis habitasse vulputate habitant netus varius rhoncus ultrices porttitor himenaeos lorem libero congue turpis parturient quisque nostra aliquet in sem curabitur eleifend accumsan faucibus per pellentesque. Nibh auctor dictum vivamus eros ex gravida hac torquent purus suscipit fames lacus sagittis condimentum morbi dui litora cras duis iaculis massa porta praesent sapien. Ultricies tortor phasellus mus erat metus nisi malesuada augue sollicitudin convallis egestas ultrices arcu luctus tempus molestie facilisis nam scelerisque feugiat. Nibh imperdiet accumsan fermentum auctor et neque blandit elementum id eget justo suscipit interdum etiam mus tempus diam cursus nunc malesuada aliquam vestibulum. Arcu facilisi curae sed mi felis commodo, sapien neque aenean nullam rutrum torquent lectus fringilla rhoncus eros elit molestie.\nAptent fringilla cubilia sed duis non eu vulputate dis efficitur per ad venenatis dictumst egestas commodo blandit. Conubia finibus curae molestie egestas interdum mollis aliquam venenatis penatibus habitant varius natoque aptent nec. Mattis hac commodo integer donec gravida himenaeos vivamus primis rhoncus nam cursus erat nibh nascetur elementum felis duis in volutpat aliquet nulla vehicula placerat est. Placerat dis est aenean laoreet convallis metus sit mi, porttitor ullamcorper risus augue commodo dictumst nisi platea cubilia maximus elit volutpat hac rutrum suspendisse. Lacinia taciti justo non ligula vivamus aliquam luctus tellus dictumst vulputate interdum per aptent a metus eu mauris hac ex montes senectus blandit proin. Proin ullamcorper habitant justo pharetra felis commodo parturient scelerisque rutrum suspendisse ad ante cubilia pulvinar est vivamus quisque imperdiet vestibulum varius aliquam enim. Mollis aliquam metus montes dapibus volutpat maecenas fermentum massa tempor condimentum rhoncus lacinia tincidunt accumsan leo nunc elementum maximus fringilla dui augue."
            }
          },
          "textBody": [
            {
              "partId": "0",
              "blobId": "cg0tuhkv1lncttirzg9p97wd7k7gezaz9fbwjir31wrcvkykbm1zkaya9imiuaxgdu",
              "size": 3814,
              "name": null,
              "type": "text/plain",
              "charset": "utf-8",
              "disposition": null,
              "cid": null,
              "language": null,
              "location": null
            }
          ],
          "htmlBody": [
            {
              "partId": "0",
              "blobId": "cg0tuhkv1lncttirzg9p97wd7k7gezaz9fbwjir31wrcvkykbm1zkaya9imiuaxgdu",
              "size": 3814,
              "name": null,
              "type": "text/plain",
              "charset": "utf-8",
              "disposition": null,
              "cid": null,
              "language": null,
              "location": null
            }
          ],
          "attachments": []
        }		  
`

const jmap_email_with_html_body = `
        {
          "id": "mleaaadcz",
          "blobId": "cdrahu0j7gjhl3jscjnzt0ursycvwn29u9uxjlrlcpeinrm2r0yz1aya9ema",
          "threadId": "dcz",
          "mailboxIds": {
            "a": true
          },
          "keywords": {},
          "size": 10556,
          "receivedAt": "2025-05-30T09:59:55Z",
          "messageId": [
            "1748599195.3428368@example.com"
          ],
          "inReplyTo": null,
          "references": null,
          "sender": null,
          "from": [
            {
              "name": "Eminent Extremely",
              "email": "eminent.extremely@example.com"
            }
          ],
          "to": [
            {
              "name": "alan",
              "email": "alan@example.com"
            }
          ],
          "cc": null,
          "bcc": null,
          "replyTo": null,
          "subject": "Et Magnis Pulvinar Congue Aliquet Tincidunt Morbi Lobortis Mattis",
          "sentAt": "2025-05-30T11:59:55+02:00",
          "hasAttachment": false,
          "preview": "Lorem ipsum dolor sit amet consectetur adipiscing elit, montes aenean lectus porttitor mauris ridiculus rutrum et inceptos torquent congue tristique dictumst nullam suspendisse. Lobortis ad per habitasse volutpat proin posuere convallis dapibus tristiqu...",
          "bodyValues": {
            "0": {
              "isEncodingProblem": false,
              "isTruncated": false,
              "value": "<p>Lorem ipsum dolor sit amet consectetur adipiscing elit, montes aenean lectus porttitor mauris ridiculus rutrum et inceptos torquent congue tristique dictumst nullam suspendisse. Lobortis ad per habitasse volutpat proin posuere convallis dapibus tristique lacinia placerat scelerisque curabitur sed aenean sodales pharetra est nisl odio sagittis platea in. Venenatis semper inceptos laoreet orci aliquam natoque magna id tempor lacus duis convallis molestie ridiculus vivamus etiam tortor ultrices blandit dictum finibus volutpat. Finibus amet donec justo lectus senectus morbi convallis a hendrerit malesuada neque nisl ad nulla per nunc praesent.</p>\n<p>Elit dolor nostra vehicula massa placerat convallis dictum natoque commodo diam, ultricies nam consequat inceptos torquent himenaeos risus eleifend scelerisque dui cursus libero nisl neque fusce montes metus proin cras donec nibh. Tempus sodales fames consectetur in integer aliquet odio maecenas est sapien risus parturient lorem aliquam viverra mattis feugiat eu platea ex tempor mi efficitur a. Ut vestibulum nibh et himenaeos taciti nisl class pretium maximus est ultrices fermentum nunc mus dapibus vel massa venenatis facilisis non nascetur leo. Scelerisque metus nisi suspendisse pharetra fames malesuada pretium dictumst, etiam potenti molestie vestibulum habitasse aenean velit ridiculus condimentum ut montes at tortor arcu curae id luctus.</p>\n<p>Nulla sollicitudin vestibulum vulputate urna sem etiam senectus turpis ac tempus, laoreet natoque metus justo dapibus purus libero fringilla aenean orci integer imperdiet duis curabitur feugiat blandit proin consequat quam velit ante. Scelerisque elit tincidunt feugiat primis risus amet ac interdum varius luctus quis dui consectetur platea conubia senectus mus efficitur mauris cubilia libero magnis egestas elementum ultricies. Elit dapibus finibus proin aliquet etiam nibh quam laoreet senectus primis a mattis vel montes massa porta dui commodo velit mi bibendum cubilia sed euismod. Posuere consequat velit mauris in sollicitudin id dolor nisl placerat magna aliquet sed metus curae. Penatibus commodo cubilia ex velit leo ultricies ipsum dignissim molestie curae lectus, integer a risus bibendum varius ornare laoreet fermentum fusce duis luctus ultrices nostra sem id nascetur dictum tempor morbi aliquet mauris. Duis posuere enim odio nisl condimentum nunc eleifend nullam primis maecenas, tellus pretium congue nascetur lacinia in lorem vel quisque lectus proin laoreet consectetur faucibus aliquet montes ad sodales commodo vestibulum. Fames odio luctus donec habitasse neque posuere purus quis penatibus netus mi lobortis suspendisse vehicula eu lorem erat libero in scelerisque leo dapibus tristique amet.</p>\n<p>Vulputate erat consectetur cras iaculis nascetur lectus pulvinar fames est ut malesuada natoque hac orci euismod rhoncus ad faucibus nostra aptent sociosqu. Leo primis dictumst libero platea nisl at mauris eu fusce penatibus, nunc maximus sodales montes facilisis pharetra ex ipsum class curae aenean parturient tortor massa morbi cras varius ut augue vivamus elementum. Lorem eget facilisi nec varius elementum mattis aliquam praesent blandit dapibus aliquet ornare montes malesuada taciti netus egestas lacus morbi. A nascetur nibh commodo sodales consequat nullam taciti risus, viverra proin quam quis elementum libero molestie fusce egestas curae augue mattis nisl montes senectus. Elementum tempus in dictum sagittis ac hac feugiat lorem efficitur consequat neque per tellus penatibus. Ut suscipit tempus nec tincidunt potenti libero luctus eleifend auctor pulvinar ultrices purus imperdiet dignissim at mus et montes phasellus maecenas hac egestas nulla porta. Placerat sed netus consectetur dis duis varius elementum convallis nostra natoque. Massa morbi ante egestas sit feugiat fusce conubia imperdiet vestibulum maximus mollis himenaeos porttitor auctor aliquet neque suscipit rhoncus viverra natoque vivamus posuere commodo arcu quis cras.</p>\n<p>Habitant sem ullamcorper euismod libero curabitur orci urna felis senectus lacus nunc congue morbi molestie adipiscing per sed pharetra magna ut arcu convallis consectetur non. Lorem vel id elementum lobortis netus scelerisque diam fames volutpat tristique congue justo penatibus bibendum sociosqu adipiscing est auctor habitasse ullamcorper cursus quis. Risus quis fermentum vehicula adipiscing erat orci, aptent proin gravida habitasse porttitor mattis ipsum praesent ligula feugiat ad efficitur integer. Lorem maecenas venenatis per suscipit accumsan aliquet penatibus faucibus facilisi facilisis sodales platea suspendisse euismod. Turpis posuere nisi ut tempor aliquet dapibus cursus ante mauris sed auctor dictum egestas nullam sapien porttitor justo pretium. Cursus lorem quisque sem leo convallis molestie etiam conubia pretium lacinia ultricies vestibulum nec sodales natoque commodo dictumst volutpat fames parturient justo cubilia augue velit purus aliquam. Nisl integer mus ultricies laoreet congue vivamus cras ultrices orci fames quis non tempor vel libero at nulla malesuada fermentum dolor lacus ornare ut sodales adipiscing eros nascetur lectus. Platea turpis libero habitasse a praesent cras sem eros hendrerit finibus integer tempor ipsum sapien in nostra litora sit montes risus iaculis class at torquent non magna suspendisse purus dictumst vulputate mi sodales curabitur scelerisque.</p>\n<p>Metus laoreet morbi erat ligula gravida non montes aliquam et bibendum tempus pharetra posuere nulla eleifend ante tortor habitant. Suscipit nisl proin natoque mollis ligula commodo scelerisque leo pellentesque per senectus adipiscing quis varius aenean curae phasellus magnis aptent felis nec nibh nisi erat lobortis auctor vehicula molestie. Fames purus velit bibendum maecenas tortor ultricies maximus convallis rhoncus inceptos per porta ipsum eu non habitant lacinia pellentesque. Ante et platea id at tempus magna orci etiam feugiat habitant conubia nascetur aenean. Purus nostra lectus lobortis etiam est lacus luctus laoreet sed ac lacinia quis at egestas class ridiculus litora eleifend urna porttitor enim.</p>\n<p>Faucibus felis integer eleifend in molestie eget platea tincidunt dui nec aliquam ultricies sodales quam porttitor facilisi potenti facilisis nisl vehicula tempus curae arcu. Magnis aliquet mi per mollis egestas porta montes ut pulvinar arcu neque adipiscing duis feugiat vel senectus quis facilisi elit nibh felis sodales ullamcorper diam sollicitudin ad. Venenatis hendrerit eget quisque sagittis facilisi quam non sociosqu curae enim, potenti augue dapibus ullamcorper auctor mi dignissim etiam viverra orci commodo laoreet inceptos pellentesque adipiscing class ac sem luctus faucibus fringilla. Urna ante in class auctor facilisi risus himenaeos, vitae malesuada dui mattis mollis aenean cras porttitor dignissim praesent egestas pretium condimentum aliquam. Arcu fringilla dictumst turpis vitae ex tempus vehicula efficitur maximus tincidunt pulvinar praesent nulla odio lacus ridiculus fames pharetra mauris ornare felis aenean penatibus taciti dignissim fusce diam orci vel. Habitant lectus primis risus nisi dolor erat senectus eros, felis varius sit phasellus quisque congue blandit bibendum ante est ligula nostra aliquet aptent magna purus. Pharetra eu hendrerit pulvinar magnis primis quisque integer in mus pellentesque suspendisse lacinia sem dictumst nisl auctor maximus platea.</p>\n<p>Nascetur tortor ac placerat facilisi integer litora sit varius duis efficitur sapien, hendrerit diam accumsan elit montes vehicula magnis consequat nostra justo parturient torquent pretium interdum a tincidunt dictum magna vel etiam ut dolor ullamcorper. Tincidunt aliquam lectus id velit class ad malesuada auctor litora consectetur aptent pharetra etiam dolor et tristique lacus.</p>\n<p>Non quis nullam urna ligula aptent curabitur odio lacus suspendisse lacinia molestie mus morbi elementum maximus interdum a purus enim sem sapien scelerisque lobortis et phasellus. Elementum vulputate vehicula posuere iaculis sodales fames urna rhoncus purus, laoreet metus ornare sem consequat mollis nibh lorem parturient adipiscing porttitor pretium placerat habitasse libero eleifend enim morbi. Massa tellus viverra nascetur leo aenean nisl vivamus malesuada at ipsum lobortis rutrum accumsan senectus dignissim elit fermentum integer praesent a purus proin faucibus aptent ad adipiscing imperdiet convallis mauris sodales. Sed iaculis mauris ut nunc fusce justo et venenatis libero litora eget aliquet penatibus gravida interdum phasellus turpis ullamcorper cubilia duis leo ex mattis vel cras donec lacinia sodales malesuada id elementum. Mus ultrices ullamcorper suspendisse nec dapibus senectus fermentum felis netus non magna congue neque bibendum dignissim ipsum aenean integer curae facilisi donec. Dolor purus nibh diam facilisis erat etiam mollis consectetur semper, vestibulum suscipit mauris egestas venenatis neque varius dignissim pulvinar ligula lobortis morbi aliquam eros nullam suspendisse orci tortor. Accumsan rutrum tempus arcu eros convallis vel natoque commodo eget diam mollis himenaeos proin placerat suspendisse duis taciti. Urna gravida a mus lacinia aliquam justo in lectus nec sed habitasse penatibus et ex massa vel commodo facilisis rutrum taciti odio torquent inceptos imperdiet sociosqu montes cursus nostra suscipit quam venenatis. Mattis nulla congue interdum gravida ornare ac sed, sagittis iaculis sem parturient netus proin maecenas dignissim rhoncus efficitur condimentum egestas dis litora. Odio imperdiet facilisi tempus ipsum donec tortor dictumst sem finibus parturient aptent molestie pretium risus sagittis pellentesque nisi litora congue cras viverra enim tempor vehicula platea.</p>\n<p>Sodales pretium egestas libero viverra lobortis interdum amet quis fames neque convallis dictumst sollicitudin eros felis nec. Nibh nisi sit nam magna elit fames habitasse sollicitudin libero lacus luctus porttitor enim conubia dolor suscipit aptent platea dictum habitant primis imperdiet taciti. Lobortis dui scelerisque feugiat venenatis vehicula tristique mi iaculis efficitur imperdiet aliquet sociosqu ipsum ornare sed gravida amet platea nisl mollis consectetur ex ac.</p>"
            }
          },
          "textBody": [
            {
              "partId": "0",
              "blobId": "chrahu0j7gjhl3jscjnzt0ursycvwn29u9uxjlrlcpeinrm2r0yz1aya9emlqaueka",
              "size": 10244,
              "name": null,
              "type": "text/html",
              "charset": "utf-8",
              "disposition": null,
              "cid": null,
              "language": null,
              "location": null
            }
          ],
          "htmlBody": [
            {
              "partId": "0",
              "blobId": "chrahu0j7gjhl3jscjnzt0ursycvwn29u9uxjlrlcpeinrm2r0yz1aya9emlqaueka",
              "size": 10244,
              "name": null,
              "type": "text/html",
              "charset": "utf-8",
              "disposition": null,
              "cid": null,
              "language": null,
              "location": null
            }
          ],
          "attachments": []
        }
`

func TestMapEmailWithTextAndHtmlBodies(t *testing.T) {
	require := require.New(t)

	var elem map[string]any
	err := json.Unmarshal([]byte(jmap_email_with_text_and_html_bodies), &elem)
	require.NoError(err)

	logger := log.NopLogger()

	email, err := mapEmail(elem, true, &logger)
	require.NoError(err)
	require.Equal("libero ad blandit rutrum lacinia consectetur sem", email.Subject)
	require.Equal("Diam egestas imperdiet non eu quam semper euismod netus venenatis ante magnis mus finibus maecenas nec cras ac commodo nascetur aliquet habitasse porta velit felis tempus ligula vulputate. Elit dolor fames neque hac nunc ornare nibh facilisis nisl finib...", email.Preview)
	require.Len(email.Bodies, 2)
	require.Contains(email.Bodies, "text/html")
	require.Equal("<p>Diam egestas imperdiet non eu quam semper euismod netus venenatis ante magnis mus finibus maecenas nec cras ac commodo nascetur aliquet habitasse porta velit felis tempus ligula vulputate. Elit dolor fames neque hac nunc ornare nibh facilisis nisl finibus magna senectus montes vulputate justo dis cras interdum. Convallis montes urna iaculis etiam mauris lorem tristique accumsan erat tincidunt posuere quis felis primis dolor a ipsum hendrerit parturient dictum pulvinar phasellus id porttitor. Etiam mi sollicitudin justo eu natoque eros malesuada nostra vulputate maximus habitant arcu fames imperdiet odio at netus eget maecenas elit parturient hendrerit nibh dui augue quisque tellus amet platea sit. Lectus risus potenti bibendum gravida fringilla sollicitudin sit enim consectetur ipsum accumsan parturient lorem varius sagittis rutrum montes vehicula nec mus hendrerit hac malesuada vel ac integer elementum.</p>\n<p>Euismod donec aliquet suspendisse mi blandit faucibus egestas adipiscing purus congue id himenaeos aenean per. Nullam habitasse est volutpat montes laoreet posuere eget suscipit, ultrices interdum mi nulla ac at eleifend praesent dis nostra massa habitant sapien integer porta consequat amet. Ut conubia amet vulputate ridiculus euismod fermentum libero auctor, mus donec eros a netus ad condimentum morbi facilisi neque tellus feugiat class erat metus inceptos.</p>\n<p>Aenean himenaeos ridiculus risus dictum taciti dui quisque penatibus interdum magnis sollicitudin commodo tempor ultrices dapibus mi tempus ullamcorper nibh volutpat justo consequat fusce amet hendrerit laoreet dignissim sit venenatis semper libero mus suscipit. Quis aptent non varius porttitor aliquam iaculis justo facilisi nostra sodales imperdiet integer odio tincidunt quisque rhoncus ullamcorper laoreet tristique dolor. Blandit fringilla adipiscing dictumst euismod magnis volutpat tortor mollis varius elementum nostra litora porttitor habitant convallis risus urna consectetur eleifend suspendisse auctor posuere. Senectus mauris purus a dis tincidunt parturient tortor proin facilisis taciti tellus egestas dui ante in turpis adipiscing lacus neque quisque sagittis tristique suscipit est vestibulum nullam. Pellentesque massa ligula lobortis habitasse rutrum finibus fermentum hac egestas augue aliquet efficitur volutpat mattis ac imperdiet malesuada id etiam turpis tempus tellus interdum quisque at pulvinar nullam proin velit dictumst. Habitant rutrum sit dignissim porta luctus aenean volutpat aliquam arcu lacus tincidunt augue mattis porttitor neque congue risus nostra ridiculus dui molestie maximus libero justo faucibus.</p>\n<p>Nisl condimentum pulvinar vulputate quam ante urna habitasse suscipit, volutpat lorem venenatis sem dignissim sapien penatibus ipsum felis faucibus eget velit sociosqu dictumst arcu viverra erat. Vitae auctor lobortis etiam ligula maecenas aptent fringilla, tempus pellentesque euismod neque sociosqu posuere curabitur venenatis dis elit inceptos ullamcorper natoque. Suspendisse elementum semper diam luctus odio fringilla sem nascetur blandit nam cubilia integer senectus in sociosqu sollicitudin nisi parturient. Ante maximus donec hac malesuada nisl quam massa nunc justo conubia fringilla tellus natoque scelerisque cubilia litora.</p>\n<p>Aliquet morbi ligula quisque dapibus ultrices eros sem malesuada lobortis massa litora vestibulum varius commodo egestas tincidunt aenean ullamcorper at duis velit auctor parturient. Feugiat natoque posuere orci rhoncus ante mus quam, quis non sapien ut purus volutpat himenaeos et senectus fermentum placerat elementum augue. Natoque id mauris vel mus molestie elementum fames hac consectetur sed platea ad eget efficitur maecenas conubia morbi justo vivamus placerat curae pretium nisi ipsum imperdiet. Velit eros volutpat efficitur pharetra natoque primis luctus nunc lacus fusce dolor sagittis porta maecenas odio rutrum dis consectetur nam metus venenatis ut. Iaculis turpis luctus per orci taciti vehicula amet ad integer, quis litora mauris praesent ullamcorper cursus faucibus at eros dictum dolor morbi mus semper senectus laoreet felis torquent. Phasellus senectus nibh ornare dui convallis orci consequat enim justo etiam himenaeos dictum velit dis magna tempor maecenas fermentum luctus morbi molestie praesent condimentum hendrerit penatibus nisl tempus.</p>", email.Bodies["text/html"])
	require.Contains(email.Bodies, "text/plain")
	require.Equal("Diam egestas imperdiet non eu quam semper euismod netus venenatis ante magnis mus finibus maecenas nec cras ac commodo nascetur aliquet habitasse porta velit felis tempus ligula vulputate. Elit dolor fames neque hac nunc ornare nibh facilisis nisl finibus magna senectus montes vulputate justo dis cras interdum. Convallis montes urna iaculis etiam mauris lorem tristique accumsan erat tincidunt posuere quis felis primis dolor a ipsum hendrerit parturient dictum pulvinar phasellus id porttitor. Etiam mi sollicitudin justo eu natoque eros malesuada nostra vulputate maximus habitant arcu fames imperdiet odio at netus eget maecenas elit parturient hendrerit nibh dui augue quisque tellus amet platea sit. Lectus risus potenti bibendum gravida fringilla sollicitudin sit enim consectetur ipsum accumsan parturient lorem varius sagittis rutrum montes vehicula nec mus hendrerit hac malesuada vel ac integer elementum.\nEuismod donec aliquet suspendisse mi blandit faucibus egestas adipiscing purus congue id himenaeos aenean per. Nullam habitasse est volutpat montes laoreet posuere eget suscipit, ultrices interdum mi nulla ac at eleifend praesent dis nostra massa habitant sapien integer porta consequat amet. Ut conubia amet vulputate ridiculus euismod fermentum libero auctor, mus donec eros a netus ad condimentum morbi facilisi neque tellus feugiat class erat metus inceptos.\nAenean himenaeos ridiculus risus dictum taciti dui quisque penatibus interdum magnis sollicitudin commodo tempor ultrices dapibus mi tempus ullamcorper nibh volutpat justo consequat fusce amet hendrerit laoreet dignissim sit venenatis semper libero mus suscipit. Quis aptent non varius porttitor aliquam iaculis justo facilisi nostra sodales imperdiet integer odio tincidunt quisque rhoncus ullamcorper laoreet tristique dolor. Blandit fringilla adipiscing dictumst euismod magnis volutpat tortor mollis varius elementum nostra litora porttitor habitant convallis risus urna consectetur eleifend suspendisse auctor posuere. Senectus mauris purus a dis tincidunt parturient tortor proin facilisis taciti tellus egestas dui ante in turpis adipiscing lacus neque quisque sagittis tristique suscipit est vestibulum nullam. Pellentesque massa ligula lobortis habitasse rutrum finibus fermentum hac egestas augue aliquet efficitur volutpat mattis ac imperdiet malesuada id etiam turpis tempus tellus interdum quisque at pulvinar nullam proin velit dictumst. Habitant rutrum sit dignissim porta luctus aenean volutpat aliquam arcu lacus tincidunt augue mattis porttitor neque congue risus nostra ridiculus dui molestie maximus libero justo faucibus.\nNisl condimentum pulvinar vulputate quam ante urna habitasse suscipit, volutpat lorem venenatis sem dignissim sapien penatibus ipsum felis faucibus eget velit sociosqu dictumst arcu viverra erat. Vitae auctor lobortis etiam ligula maecenas aptent fringilla, tempus pellentesque euismod neque sociosqu posuere curabitur venenatis dis elit inceptos ullamcorper natoque. Suspendisse elementum semper diam luctus odio fringilla sem nascetur blandit nam cubilia integer senectus in sociosqu sollicitudin nisi parturient. Ante maximus donec hac malesuada nisl quam massa nunc justo conubia fringilla tellus natoque scelerisque cubilia litora.\nAliquet morbi ligula quisque dapibus ultrices eros sem malesuada lobortis massa litora vestibulum varius commodo egestas tincidunt aenean ullamcorper at duis velit auctor parturient. Feugiat natoque posuere orci rhoncus ante mus quam, quis non sapien ut purus volutpat himenaeos et senectus fermentum placerat elementum augue. Natoque id mauris vel mus molestie elementum fames hac consectetur sed platea ad eget efficitur maecenas conubia morbi justo vivamus placerat curae pretium nisi ipsum imperdiet. Velit eros volutpat efficitur pharetra natoque primis luctus nunc lacus fusce dolor sagittis porta maecenas odio rutrum dis consectetur nam metus venenatis ut. Iaculis turpis luctus per orci taciti vehicula amet ad integer, quis litora mauris praesent ullamcorper cursus faucibus at eros dictum dolor morbi mus semper senectus laoreet felis torquent. Phasellus senectus nibh ornare dui convallis orci consequat enim justo etiam himenaeos dictum velit dis magna tempor maecenas fermentum luctus morbi molestie praesent condimentum hendrerit penatibus nisl tempus.", email.Bodies["text/plain"])
	require.Equal("1748595212.4933355@example.com", email.MessageId)
	require.False(email.HasAttachments)
	require.Equal("cby92nwhy2pswwygvnavdnv0zc3kffdafeauqko2lyw1qvtnjhztwaya72ma", email.BlobId)

}

func TestMapEmailWithHtmlBody(t *testing.T) {
	require := require.New(t)

	var elem map[string]any
	err := json.Unmarshal([]byte(jmap_email_with_html_body), &elem)
	require.NoError(err)

	logger := log.NopLogger()

	email, err := mapEmail(elem, true, &logger)
	require.NoError(err)
	require.Len(email.Bodies, 1)
	require.Contains(email.Bodies, "text/html")
	require.Equal("<p>Lorem ipsum dolor sit amet consectetur adipiscing elit, montes aenean lectus porttitor mauris ridiculus rutrum et inceptos torquent congue tristique dictumst nullam suspendisse. Lobortis ad per habitasse volutpat proin posuere convallis dapibus tristique lacinia placerat scelerisque curabitur sed aenean sodales pharetra est nisl odio sagittis platea in. Venenatis semper inceptos laoreet orci aliquam natoque magna id tempor lacus duis convallis molestie ridiculus vivamus etiam tortor ultrices blandit dictum finibus volutpat. Finibus amet donec justo lectus senectus morbi convallis a hendrerit malesuada neque nisl ad nulla per nunc praesent.</p>\n<p>Elit dolor nostra vehicula massa placerat convallis dictum natoque commodo diam, ultricies nam consequat inceptos torquent himenaeos risus eleifend scelerisque dui cursus libero nisl neque fusce montes metus proin cras donec nibh. Tempus sodales fames consectetur in integer aliquet odio maecenas est sapien risus parturient lorem aliquam viverra mattis feugiat eu platea ex tempor mi efficitur a. Ut vestibulum nibh et himenaeos taciti nisl class pretium maximus est ultrices fermentum nunc mus dapibus vel massa venenatis facilisis non nascetur leo. Scelerisque metus nisi suspendisse pharetra fames malesuada pretium dictumst, etiam potenti molestie vestibulum habitasse aenean velit ridiculus condimentum ut montes at tortor arcu curae id luctus.</p>\n<p>Nulla sollicitudin vestibulum vulputate urna sem etiam senectus turpis ac tempus, laoreet natoque metus justo dapibus purus libero fringilla aenean orci integer imperdiet duis curabitur feugiat blandit proin consequat quam velit ante. Scelerisque elit tincidunt feugiat primis risus amet ac interdum varius luctus quis dui consectetur platea conubia senectus mus efficitur mauris cubilia libero magnis egestas elementum ultricies. Elit dapibus finibus proin aliquet etiam nibh quam laoreet senectus primis a mattis vel montes massa porta dui commodo velit mi bibendum cubilia sed euismod. Posuere consequat velit mauris in sollicitudin id dolor nisl placerat magna aliquet sed metus curae. Penatibus commodo cubilia ex velit leo ultricies ipsum dignissim molestie curae lectus, integer a risus bibendum varius ornare laoreet fermentum fusce duis luctus ultrices nostra sem id nascetur dictum tempor morbi aliquet mauris. Duis posuere enim odio nisl condimentum nunc eleifend nullam primis maecenas, tellus pretium congue nascetur lacinia in lorem vel quisque lectus proin laoreet consectetur faucibus aliquet montes ad sodales commodo vestibulum. Fames odio luctus donec habitasse neque posuere purus quis penatibus netus mi lobortis suspendisse vehicula eu lorem erat libero in scelerisque leo dapibus tristique amet.</p>\n<p>Vulputate erat consectetur cras iaculis nascetur lectus pulvinar fames est ut malesuada natoque hac orci euismod rhoncus ad faucibus nostra aptent sociosqu. Leo primis dictumst libero platea nisl at mauris eu fusce penatibus, nunc maximus sodales montes facilisis pharetra ex ipsum class curae aenean parturient tortor massa morbi cras varius ut augue vivamus elementum. Lorem eget facilisi nec varius elementum mattis aliquam praesent blandit dapibus aliquet ornare montes malesuada taciti netus egestas lacus morbi. A nascetur nibh commodo sodales consequat nullam taciti risus, viverra proin quam quis elementum libero molestie fusce egestas curae augue mattis nisl montes senectus. Elementum tempus in dictum sagittis ac hac feugiat lorem efficitur consequat neque per tellus penatibus. Ut suscipit tempus nec tincidunt potenti libero luctus eleifend auctor pulvinar ultrices purus imperdiet dignissim at mus et montes phasellus maecenas hac egestas nulla porta. Placerat sed netus consectetur dis duis varius elementum convallis nostra natoque. Massa morbi ante egestas sit feugiat fusce conubia imperdiet vestibulum maximus mollis himenaeos porttitor auctor aliquet neque suscipit rhoncus viverra natoque vivamus posuere commodo arcu quis cras.</p>\n<p>Habitant sem ullamcorper euismod libero curabitur orci urna felis senectus lacus nunc congue morbi molestie adipiscing per sed pharetra magna ut arcu convallis consectetur non. Lorem vel id elementum lobortis netus scelerisque diam fames volutpat tristique congue justo penatibus bibendum sociosqu adipiscing est auctor habitasse ullamcorper cursus quis. Risus quis fermentum vehicula adipiscing erat orci, aptent proin gravida habitasse porttitor mattis ipsum praesent ligula feugiat ad efficitur integer. Lorem maecenas venenatis per suscipit accumsan aliquet penatibus faucibus facilisi facilisis sodales platea suspendisse euismod. Turpis posuere nisi ut tempor aliquet dapibus cursus ante mauris sed auctor dictum egestas nullam sapien porttitor justo pretium. Cursus lorem quisque sem leo convallis molestie etiam conubia pretium lacinia ultricies vestibulum nec sodales natoque commodo dictumst volutpat fames parturient justo cubilia augue velit purus aliquam. Nisl integer mus ultricies laoreet congue vivamus cras ultrices orci fames quis non tempor vel libero at nulla malesuada fermentum dolor lacus ornare ut sodales adipiscing eros nascetur lectus. Platea turpis libero habitasse a praesent cras sem eros hendrerit finibus integer tempor ipsum sapien in nostra litora sit montes risus iaculis class at torquent non magna suspendisse purus dictumst vulputate mi sodales curabitur scelerisque.</p>\n<p>Metus laoreet morbi erat ligula gravida non montes aliquam et bibendum tempus pharetra posuere nulla eleifend ante tortor habitant. Suscipit nisl proin natoque mollis ligula commodo scelerisque leo pellentesque per senectus adipiscing quis varius aenean curae phasellus magnis aptent felis nec nibh nisi erat lobortis auctor vehicula molestie. Fames purus velit bibendum maecenas tortor ultricies maximus convallis rhoncus inceptos per porta ipsum eu non habitant lacinia pellentesque. Ante et platea id at tempus magna orci etiam feugiat habitant conubia nascetur aenean. Purus nostra lectus lobortis etiam est lacus luctus laoreet sed ac lacinia quis at egestas class ridiculus litora eleifend urna porttitor enim.</p>\n<p>Faucibus felis integer eleifend in molestie eget platea tincidunt dui nec aliquam ultricies sodales quam porttitor facilisi potenti facilisis nisl vehicula tempus curae arcu. Magnis aliquet mi per mollis egestas porta montes ut pulvinar arcu neque adipiscing duis feugiat vel senectus quis facilisi elit nibh felis sodales ullamcorper diam sollicitudin ad. Venenatis hendrerit eget quisque sagittis facilisi quam non sociosqu curae enim, potenti augue dapibus ullamcorper auctor mi dignissim etiam viverra orci commodo laoreet inceptos pellentesque adipiscing class ac sem luctus faucibus fringilla. Urna ante in class auctor facilisi risus himenaeos, vitae malesuada dui mattis mollis aenean cras porttitor dignissim praesent egestas pretium condimentum aliquam. Arcu fringilla dictumst turpis vitae ex tempus vehicula efficitur maximus tincidunt pulvinar praesent nulla odio lacus ridiculus fames pharetra mauris ornare felis aenean penatibus taciti dignissim fusce diam orci vel. Habitant lectus primis risus nisi dolor erat senectus eros, felis varius sit phasellus quisque congue blandit bibendum ante est ligula nostra aliquet aptent magna purus. Pharetra eu hendrerit pulvinar magnis primis quisque integer in mus pellentesque suspendisse lacinia sem dictumst nisl auctor maximus platea.</p>\n<p>Nascetur tortor ac placerat facilisi integer litora sit varius duis efficitur sapien, hendrerit diam accumsan elit montes vehicula magnis consequat nostra justo parturient torquent pretium interdum a tincidunt dictum magna vel etiam ut dolor ullamcorper. Tincidunt aliquam lectus id velit class ad malesuada auctor litora consectetur aptent pharetra etiam dolor et tristique lacus.</p>\n<p>Non quis nullam urna ligula aptent curabitur odio lacus suspendisse lacinia molestie mus morbi elementum maximus interdum a purus enim sem sapien scelerisque lobortis et phasellus. Elementum vulputate vehicula posuere iaculis sodales fames urna rhoncus purus, laoreet metus ornare sem consequat mollis nibh lorem parturient adipiscing porttitor pretium placerat habitasse libero eleifend enim morbi. Massa tellus viverra nascetur leo aenean nisl vivamus malesuada at ipsum lobortis rutrum accumsan senectus dignissim elit fermentum integer praesent a purus proin faucibus aptent ad adipiscing imperdiet convallis mauris sodales. Sed iaculis mauris ut nunc fusce justo et venenatis libero litora eget aliquet penatibus gravida interdum phasellus turpis ullamcorper cubilia duis leo ex mattis vel cras donec lacinia sodales malesuada id elementum. Mus ultrices ullamcorper suspendisse nec dapibus senectus fermentum felis netus non magna congue neque bibendum dignissim ipsum aenean integer curae facilisi donec. Dolor purus nibh diam facilisis erat etiam mollis consectetur semper, vestibulum suscipit mauris egestas venenatis neque varius dignissim pulvinar ligula lobortis morbi aliquam eros nullam suspendisse orci tortor. Accumsan rutrum tempus arcu eros convallis vel natoque commodo eget diam mollis himenaeos proin placerat suspendisse duis taciti. Urna gravida a mus lacinia aliquam justo in lectus nec sed habitasse penatibus et ex massa vel commodo facilisis rutrum taciti odio torquent inceptos imperdiet sociosqu montes cursus nostra suscipit quam venenatis. Mattis nulla congue interdum gravida ornare ac sed, sagittis iaculis sem parturient netus proin maecenas dignissim rhoncus efficitur condimentum egestas dis litora. Odio imperdiet facilisi tempus ipsum donec tortor dictumst sem finibus parturient aptent molestie pretium risus sagittis pellentesque nisi litora congue cras viverra enim tempor vehicula platea.</p>\n<p>Sodales pretium egestas libero viverra lobortis interdum amet quis fames neque convallis dictumst sollicitudin eros felis nec. Nibh nisi sit nam magna elit fames habitasse sollicitudin libero lacus luctus porttitor enim conubia dolor suscipit aptent platea dictum habitant primis imperdiet taciti. Lobortis dui scelerisque feugiat venenatis vehicula tristique mi iaculis efficitur imperdiet aliquet sociosqu ipsum ornare sed gravida amet platea nisl mollis consectetur ex ac.</p>", email.Bodies["text/html"])
}

func TestMapEmailWithTextBody(t *testing.T) {
	require := require.New(t)

	var elem map[string]any
	err := json.Unmarshal([]byte(jmap_email_with_text_body), &elem)
	require.NoError(err)

	logger := log.NopLogger()

	email, err := mapEmail(elem, true, &logger)
	require.NoError(err)
	require.Len(email.Bodies, 1)
	require.Contains(email.Bodies, "text/plain")
	require.Equal("Et magnis pulvinar congue aliquet tincidunt morbi lobortis mattis mus litora malesuada fringilla varius ullamcorper parturient fames accumsan faucibus erat. Magna id est cras eu a netus orci ridiculus lobortis urna dis ipsum at fermentum mi lacinia quis fames. Cursus ipsum gravida libero ultricies pretium montes rutrum suscipit tempor hac dapibus senectus commodo elementum leo nullam auctor litora pulvinar finibus. Ad nulla torquent quis mollis phasellus sodales aliquet lacinia varius, adipiscing enim habitant et netus egestas eu tempor malesuada mattis hac fusce integer diam inceptos venenatis turpis. Sem senectus aptent non dolor hendrerit magna mauris facilisis justo quam fringilla cursus gravida praesent malesuada taciti odio etiam magnis nostra vivamus. Tempus fames faucibus massa rutrum sit habitant morbi curabitur integer erat et condimentum tincidunt tempor libero vulputate maecenas rhoncus turpis congue a luctus aenean tristique lacinia cursus est fusce non mollis justo euismod facilisis egestas.\nAuctor maecenas vestibulum aenean accumsan eros ex potenti sociosqu, fusce sapien quis faucibus aliquam vivamus tristique hendrerit in per fermentum cras sodales curabitur scelerisque. Finibus metus adipiscing taciti eget rutrum vitae arcu torquent, dignissim at nibh venenatis facilisis molestie erat augue massa feugiat aliquam sollicitudin elementum cursus in. Est neque cras aenean felis justo euismod adipiscing magnis sagittis ut massa aliquet malesuada laoreet velit purus suspendisse bibendum pharetra litora ultrices diam ullamcorper volutpat venenatis egestas. Non laoreet eu interdum sodales phasellus morbi risus maecenas parturient auctor senectus urna ornare faucibus sociosqu habitant nisi cubilia viverra diam fames condimentum tempor scelerisque iaculis lacus elit feugiat adipiscing vivamus. Euismod volutpat gravida fames nascetur ridiculus iaculis habitasse vulputate habitant netus varius rhoncus ultrices porttitor himenaeos lorem libero congue turpis parturient quisque nostra aliquet in sem curabitur eleifend accumsan faucibus per pellentesque. Nibh auctor dictum vivamus eros ex gravida hac torquent purus suscipit fames lacus sagittis condimentum morbi dui litora cras duis iaculis massa porta praesent sapien. Ultricies tortor phasellus mus erat metus nisi malesuada augue sollicitudin convallis egestas ultrices arcu luctus tempus molestie facilisis nam scelerisque feugiat. Nibh imperdiet accumsan fermentum auctor et neque blandit elementum id eget justo suscipit interdum etiam mus tempus diam cursus nunc malesuada aliquam vestibulum. Arcu facilisi curae sed mi felis commodo, sapien neque aenean nullam rutrum torquent lectus fringilla rhoncus eros elit molestie.\nAptent fringilla cubilia sed duis non eu vulputate dis efficitur per ad venenatis dictumst egestas commodo blandit. Conubia finibus curae molestie egestas interdum mollis aliquam venenatis penatibus habitant varius natoque aptent nec. Mattis hac commodo integer donec gravida himenaeos vivamus primis rhoncus nam cursus erat nibh nascetur elementum felis duis in volutpat aliquet nulla vehicula placerat est. Placerat dis est aenean laoreet convallis metus sit mi, porttitor ullamcorper risus augue commodo dictumst nisi platea cubilia maximus elit volutpat hac rutrum suspendisse. Lacinia taciti justo non ligula vivamus aliquam luctus tellus dictumst vulputate interdum per aptent a metus eu mauris hac ex montes senectus blandit proin. Proin ullamcorper habitant justo pharetra felis commodo parturient scelerisque rutrum suspendisse ad ante cubilia pulvinar est vivamus quisque imperdiet vestibulum varius aliquam enim. Mollis aliquam metus montes dapibus volutpat maecenas fermentum massa tempor condimentum rhoncus lacinia tincidunt accumsan leo nunc elementum maximus fringilla dui augue.", email.Bodies["text/plain"])
}
