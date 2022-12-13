package facilities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/diwise/context-broker/pkg/ngsild"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/context-broker/pkg/test"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

var response = `{"type":"FeatureCollection","features":[
	{"id":1545,"type":"Feature",
	"properties":{
		"name":"Lillsjöns vinterbad","type":"Strandbad","created":"2020-06-04 14:26:58","updated":"2020-12-02 08:46:56","published":true,
		"fields":[
			{"id":153,"name":"Allmänt tillgänglig","type":"DROPDOWN","value":"Hela dygnet"},{"id":29,"name":"Sandstrand","type":"TOGGLE","value":"Nej"},{"id":30,"name":"Bergsstrand","type":"TOGGLE","value":"Nej"},{"id":33,"name":"Långgrunt","type":"TOGGLE","value":"Nej"},{"id":154,"name":"Bokningsbar","type":"TOGGLE","value":"Nej"},{"id":1,"name":"Beskrivning","type":"FREETEXT","value":"En beskrivning om stranden"},{"id":180,"name":"Kontakt länk","type":"FREETEXT","value":"https:\/\/www.facebook.com\/Badarna\/"},{"id":186,"name":"Felanmälan telefon","type":"FREETEXT","value":"060-XX XX XX"},{"id":187,"name":"Felanmälan e-post","type":"FREETEXT","value":"felanmelan@dev.null"},{"id":230,"name":"Temperatursensor","type":"FREETEXT","value":"sk-elt-temp-01"}
			]
	},
	"geometry":{
		"type":"MultiPolygon",
		"coordinates":[
			[
				[
					[17.472639624581532,62.43515222128755],
					[17.473786216868415,62.43536925652586],
					[17.474885857241564,62.43543825033344],
					[17.475474288890823,62.43457483981894],
					[17.47433409463916,62.43422493303495],
					[17.474073693177655,62.43422553227232],
					[17.473565135906316,62.4344799858447],
					[17.47299514306735,62.43493669748255],
					[17.472639624581532,62.43515222128755]
				]
			]
		]}
	},
	{
		"id":703,
		"type":"Feature",
		"properties":{
			"name":"Hotellslingan 5 km",
			"type":"Motionsspår",
			"created":"2019-04-05 12:39:34",
			"updated":"2021-12-11 08:14:31",
			"published":true,
			"owner":{
				"organizationID":36,
				"name":"Sundsvalls kommun Friluftsenheten"
			},
			"manager":{
				"organizationID":88,
				"name":"Sundsvalls kommun Friluftsenheten"
			},
			"fields":[
				{"id":102,"name":"Öppen","type":"TOGGLE","value":"Ja"},
				{"id":103,"name":"Belysning","type":"TOGGLE","value":"Ja"},
				{"id":108,"name":"Tillgänglighetsanpassad","type":"TOGGLE","value":"Nej"},
				{"id":250,"name":"Preparerad skidled klassiskt","type":"TOGGLE","value":"Ja"},
				{"id":251,"name":"Preparerad skidled skate","type":"TOGGLE","value":"Ja"},
				{"id":111,"name":"Statusdatum","type":"DATE","value":"2019-04-05"},
				{"id":274,"name":"Led","type":"COMBINEDTRAIL","referencedObjects":[
					{"objectID":2113,"fieldID":262,"direction":"NORMAL"},
					{"objectID":2114,"fieldID":262,"direction":"NORMAL"},
					{"objectID":2115,"fieldID":262,"direction":"NORMAL"},
					{"objectID":2116,"fieldID":262,"direction":"NORMAL"},{"objectID":2117,"fieldID":262,"direction":"NORMAL"},{"objectID":2118,"fieldID":262,"direction":"NORMAL"},{"objectID":2121,"fieldID":262,"direction":"NORMAL"},{"objectID":2123,"fieldID":262,"direction":"NORMAL"},{"objectID":2124,"fieldID":262,"direction":"NORMAL"},{"objectID":2125,"fieldID":262,"direction":"NORMAL"},{"objectID":2126,"fieldID":262,"direction":"NORMAL"},{"objectID":2150,"fieldID":262,"direction":"NORMAL"},{"objectID":2151,"fieldID":262,"direction":"NORMAL"},{"objectID":2152,"fieldID":262,"direction":"NORMAL"},{"objectID":2105,"fieldID":262,"direction":"NORMAL"},{"objectID":2106,"fieldID":262,"direction":"NORMAL"},{"objectID":2107,"fieldID":262,"direction":"NORMAL"},{"objectID":2108,"fieldID":262,"direction":"NORMAL"},{"objectID":2109,"fieldID":262,"direction":"NORMAL"},{"objectID":2110,"fieldID":262,"direction":"NORMAL"},{"objectID":2111,"fieldID":262,"direction":"NORMAL"},{"objectID":2112,"fieldID":262,"direction":"NORMAL"},{"objectID":2155,"fieldID":262,"direction":"NORMAL"},{"objectID":2156,"fieldID":262,"direction":"NORMAL"},{"objectID":2157,"fieldID":262,"direction":"NORMAL"},{"objectID":2160,"fieldID":262,"direction":"NORMAL"},{"objectID":2161,"fieldID":262,"direction":"NORMAL"},{"objectID":2128,"fieldID":262,"direction":"NORMAL"},{"objectID":2129,"fieldID":262,"direction":"NORMAL"},{"objectID":2130,"fieldID":262,"direction":"NORMAL"},{"objectID":2132,"fieldID":262,"direction":"NORMAL"},{"objectID":2133,"fieldID":262,"direction":"NORMAL"},{"objectID":2134,"fieldID":262,"direction":"NORMAL"},{"objectID":2135,"fieldID":262,"direction":"NORMAL"}
					]},
				{"id":109,"name":"Svårighet","type":"DROPDOWN","value":"Medelsvår"},
				{"id":112,"name":"Status","type":"DROPDOWN","value":"Gott skick"},
				{"id":125,"name":"Underlag","type":"DROPDOWN","value":"Grus"},
				{"id":134,"name":"Ledgrupp","type":"DROPDOWN","value":"Motionsspår Södra spårområdet"},
				{"id":104,"name":"Avgift","type":"FREETEXT","value":"Ja, vintertid"},
				{"id":110,"name":"Beskrivning","type":"FREETEXT","value":"Motionsspår med grusbeläggning. Bjuder på en lång och jobbig pkbacke, med skön lutning, samt vidunderlig utsikt över Sundsvalls hamn och stad."},
				{"id":99,"name":"Längd (meter)","type":"INTEGER","value":4700}
			]},
			"geometry":{
				"type":"LineString",
				"coordinates":[
					[17.308707161238566,62.36635873125322],[17.30876459011519,62.36642793916341],[17.30877068643981,62.36653082743534],[17.308721385538732,62.36660862404391],[17.308607388457748,62.36666251579953],[17.308441362274635,62.36669416477839],[17.308383278787474,62.366693887023146],[17.306905591601655,62.36658642682947],[17.306088435388467,62.36639658586869],[17.305201661441256,62.36618008501608],[17.30502861396609,62.366122074745334],[17.30502861396609,62.366122074745334],[17.304896742172204,62.36602282594829],[17.304828502603023,62.36597369228551],[17.304691882507868,62.36592057164345],[17.30449499509765,62.36586779494021],[17.30449499509765,62.36586779494021],[17.304302290990343,62.36583803887086],[17.304129945488526,62.36580719093905],[17.30410319227664,62.36580315495848],[17.30395491061922,62.3657770254668],[17.30379513209931,62.36575769825618],[17.30344527500855,62.36572204518083],[17.30344527500855,62.36572204518083],[17.30319332372897,62.36568077306101],[17.303047988855617,62.36562508574742],[17.302889490653907,62.365538743599004],[17.302651749182843,62.36534695647898],[17.30241674122415,62.365163692099486],[17.30235195608249,62.36511443901263],[17.30234202221011,62.36507963335273],[17.30234202221011,62.36507963335273],[17.302348877252165,62.36497025417699],[17.302388693902092,62.36490612148659],[17.30256109730512,62.364749694645496],[17.30256109730512,62.364749694645496],[17.30273305732357,62.36454799197029],[17.30285380445705,62.36439857302913],[17.302945357263944,62.36432913194809],[17.303097886569986,62.36428118126726],[17.303097886569986,62.36428118126726],[17.303135432948952,62.36427859402063],[17.303292660498347,62.364283964210536],[17.30329790958957,62.36428730152742],[17.303378302699496,62.36429160991189],[17.30338297537209,62.36429262373639],[17.303734168801444,62.364325551564534],[17.30405328389741,62.364341157801924],[17.30414951110019,62.364340366907705],[17.304290475783098,62.36433975503243],[17.304290475783098,62.36433975503243],[17.304628094395955,62.36427526027416],[17.304856745492167,62.3641877431722],[17.305138756423418,62.364016643122014],[17.305383481198334,62.36390889432864],[17.30572189078595,62.363845935931266],[17.306122995540363,62.363808930759284],[17.30632898676579,62.36378131791209],[17.3064825547944,62.36374494615831],[17.30663280604355,62.36370224116779],[17.306963321588096,62.363673714623445],[17.307091497583578,62.36366898761213],[17.307221809672136,62.36367196226756],[17.307221809672136,62.36367196226756],[17.30741269326056,62.36370398436905],[17.307598407844075,62.363756116086726],[17.307867041930482,62.363875234380885],[17.30814905921241,62.36406880752717],[17.308573099170246,62.36422471925899],[17.309016784393084,62.36438904672341],[17.309127462347238,62.36443963010415],[17.309127462347238,62.36443963010415],[17.309302742214953,62.36447571762598],[17.309507760141837,62.3645145680018],[17.3095860066008,62.36452465746457],[17.30978412940442,62.364521244127396],[17.30990118084215,62.36451183913834],[17.310031038393813,62.364491270547674],[17.310079449228464,62.36448360254857],[17.310178268586643,62.36442462327386],[17.3103275014485,62.36436760359434],[17.3105034398026,62.36433438441099],[17.310699656217604,62.36434670155311],[17.310827015024444,62.364368565228226],[17.310943342280506,62.36442038923558],[17.311170698286087,62.364535905425186],[17.311252109449107,62.36458271637739],[17.311252109449107,62.36458271637739],[17.311430813071908,62.36466136733373],[17.311675200179685,62.364754458098275],[17.311675200179685,62.364754458098275],[17.31282944428762,62.364490016228544],[17.31301572312033,62.364463091205906],[17.313232509151142,62.36443983514619],[17.313535822951565,62.36437300936384],[17.31391595844441,62.364260066546045],[17.31391595844441,62.364260066546045],[17.313998753971198,62.36425143036831],[17.314164659138097,62.36425638595836],[17.31426026621841,62.364275988021674],[17.314248713092432,62.36425646313279],[17.314331897849893,62.36425556220183],[17.314440515360687,62.36424374798666],[17.314547029709207,62.364223316294414],[17.314645276184695,62.36419803353539],[17.31472358567023,62.364151425754],[17.31485455229034,62.36405658272998],[17.31498570133707,62.36401649962371],[17.315123254764092,62.364006577009896],[17.31528479030903,62.364005526074365],[17.31537744336748,62.364010653273354],[17.31548304121338,62.364008015859575],[17.315564645597632,62.36399253431056],[17.31562565016897,62.363974760796204],[17.31562565016897,62.363974760796204],[17.315608378968765,62.36387725743011],[17.315677640411764,62.36377881602043],[17.315798385293828,62.36372004962784],[17.316066619509048,62.36364243076416],[17.31637253692994,62.36357590718918],[17.316750300053954,62.36346262892449],[17.3169072619273,62.363386222727506],[17.31700096483305,62.363281766790955],[17.317132029896666,62.36318264676178],[17.31727663710272,62.36310638142928],[17.317556345971454,62.36301140365333],[17.317628222002952,62.36296465608158],[17.317685701481555,62.362877831520336],[17.317681066540715,62.36278593981667],[17.317601370186956,62.36267762422864],[17.31754671530627,62.36257477818903],[17.317527669370758,62.36244280967177],[17.31756989962857,62.362298693237534],[17.317629100766073,62.36224633458905],[17.31779784261057,62.36215830939597],[17.31811493377235,62.36206866596348],[17.318405854297264,62.36195057730499],[17.31875771000451,62.361814587393184],[17.31887755536715,62.36173858205212],[17.319034808467624,62.36166791880201],[17.319498224655955,62.361536463418325],[17.319974570431533,62.36139018350123],[17.32010764686339,62.361357437089886],[17.320708789460692,62.36125895739428],[17.32102731447321,62.36119803480018],[17.32121013382611,62.361144322322815],[17.321635269079867,62.36099029610588],[17.321818421974765,62.36094230809729],[17.321818421974765,62.36094230809729],[17.321984996137203,62.36105540994702],[17.322238967534403,62.361184790963826],[17.322581765324045,62.36135919352776],[17.32285985455249,62.36147684300659],[17.323027254616736,62.36160718693638],[17.323428388468752,62.36220043830085],[17.32366605608134,62.36249665951174],[17.32429830763016,62.36301268879543],[17.324468052175085,62.36318897957891],[17.324548342950308,62.363308781072185],[17.32459241993049,62.36344621562721],[17.324547347418402,62.3635328990791],[17.32439303287404,62.3636610055868],[17.32427376665775,62.36374849010415],[17.324252816289494,62.363823435219004],[17.324306348809483,62.36390329566818],[17.32441021195153,62.36399984614901],[17.324501983719472,62.364102284118694],[17.32453254416536,62.36421687895377],[17.324528038256283,62.36437207413279],[17.32447904101548,62.36462545074578],[17.324507256527777,62.364694097101165],[17.324585513312638,62.36477370251693],[17.324750624768196,62.364858095425696],[17.32509229831736,62.36500951740006],[17.325158204507876,62.36508925713989],[17.32520039094369,62.36543357356673],[17.325308066437803,62.365604798721336],[17.325315641766316,62.36575411365031],[17.325282323081016,62.36582917927061],[17.325197827352074,62.36587032588341],[17.325111276395788,62.365871269378104],[17.324700346665228,62.365818284286846],[17.32450019331452,62.36577449858484],[17.324326202785418,62.36575915588967],[17.32407835266475,62.36575035889748],[17.323770420906097,62.36577669996676],[17.323463927957192,62.36583175301139],[17.323208057840287,62.36590924125294],[17.32311178745947,62.36596200464698],[17.323077881275978,62.366025586206185],[17.32305781407011,62.366117745430074],[17.323060723578536,62.3661751776448],[17.323100142850528,62.36622072253072],[17.323189006908983,62.36626572187957],[17.32333855282592,62.36628708072683],[17.323669899951575,62.36630024486383],[17.323797200861204,62.366305073277466],[17.323984405557322,62.3663375111669],[17.32420959175784,62.366386780724966],[17.324356644841565,62.36642228381108],[17.32446008326545,62.366447256847984],[17.324748817588205,62.366530310016714],[17.325084407869237,62.36656112926954],[17.32522218558481,62.3665941042786],[17.32522218558481,62.3665941042786],[17.32537257755898,62.36663269019239],[17.325512088822045,62.36670012292355],[17.325614784549366,62.366773705528665],[17.32566323703879,62.36693823019339],[17.32569672604738,62.36716931089037],[17.32575783170071,62.36739850022075],[17.325763369761546,62.367507619102085],[17.325693555913347,62.36759457989615],[17.32560905545113,62.36763571952003],[17.32545035300543,62.36767767536209],[17.325290776340488,62.367702409328366],[17.325107333443942,62.367744627240434],[17.324949503688984,62.36780381142949],[17.324757784168224,62.36792657721641],[17.324564897600002,62.36802636120031],[17.32446975677441,62.36810210017928],[17.324423803436723,62.368171554625604],[17.32416313286501,62.36839850866778],[17.324010247226,62.368555319973765],[17.32385383932682,62.36864322335841],[17.323695122768665,62.36868518476214],[17.323571470814702,62.36868653119251],[17.32344099677539,62.368682187838985],[17.323308317797533,62.36862043479652],[17.323208010522844,62.36849733544267],[17.32306885823849,62.36828974825435],[17.32306885823849,62.36828974825435],[17.322988575332555,62.368169945844265],[17.322924996122214,62.36813616100898],[17.322798450989453,62.3680800744468],[17.32274606303396,62.36802318797911],[17.3226893423249,62.36788014197699],[17.32247331822224,62.36776757226236],[17.322273139306226,62.36772378240403],[17.322099097678283,62.36770845972899],[17.321948032672843,62.367703117629254],[17.32175530717847,62.367636455196205],[17.3217498153526,62.36763446992691],[17.321565430449304,62.36756061222269],[17.32148955747765,62.367504491487814],[17.321338044491945,62.36742771266188],[17.321256205281234,62.3673489904955],[17.321229383193,62.3673017180274],[17.321218654367176,62.36722707714743],[17.321199109666296,62.367157363023885],[17.321122094829256,62.36707281873924],[17.32106040403573,62.36702554584714],[17.320987984383432,62.3669807609312],[17.32084046289504,62.366932243858315],[17.320657399548296,62.366863134184904],[17.320544190912845,62.36681508072538],[17.32038448736405,62.36674004084613],[17.320305362181212,62.366713294050875],[17.320212825969485,62.36668841329459],[17.320133700809208,62.36665544625038],[17.320069327788417,62.3666187470495],[17.319910584497407,62.36650859548437],[17.319748535419986,62.366390777209034],[17.31931965038274,62.36611055181497],[17.31924991294453,62.36608069429371],[17.319172128874328,62.36605456895026],[17.319030933450705,62.36602125449872],[17.31893743559899,62.3660010741096],[17.31885965152395,62.36597494869014],[17.318654462543943,62.36588288750993],[17.31855119748596,62.36585427383982],[17.318438544724348,62.36583250254565],[17.318206533637735,62.3658038888302],[17.318109974109337,62.36578149546811],[17.31796513482999,62.36571680344042],[17.317820295541125,62.36566206391761],[17.31765936299831,62.36562971779353],[17.317487701625343,62.3656197651312],[17.31731067583447,62.365614788801],[17.317132451181426,62.36563236921379],[17.317132451181426,62.36563236921379],[17.31704781934093,62.36564215861062],[17.316849335871616,62.36567699288659],[17.31659720821925,62.36570933896099],[17.316227456353374,62.3657462209776],[17.31609831735031,62.36575910208514],[17.315964206892314,62.365783983621384],[17.315835460868005,62.36582877032418],[17.315685257169807,62.3659083909717],[17.31546531602215,62.36598303513178],[17.31521531335754,62.366080438885426],[17.315069057268047,62.36613759481611],[17.314928403958834,62.36621086557273],[17.314600502549784,62.36638527709048],[17.31445412321788,62.36646199742176],[17.31433610603017,62.36648687836915],[17.31424634869557,62.3664997681627],[17.31424634869557,62.3664997681627],[17.314175173490188,62.366510515251406],[17.31403301640964,62.3665366402034],[17.31383587405272,62.3666206131163],[17.313744678943614,62.36665047008739],[17.31367896482767,62.366659178373084],[17.313617487632943,62.3666579902998],[17.313292726718487,62.366643627871504],[17.313260046520348,62.36663786398343],[17.31328312983948,62.36657122012164],[17.31328312983948,62.36657122012164],[17.31316715086066,62.36644810726454],[17.313112944736346,62.366404458924706],[17.31302894668589,62.36635957472811],[17.31289713341361,62.36630544684335],[17.31289713341361,62.36630544684335],[17.312640353312254,62.3663693856068],[17.31254886731595,62.366370375273824],[17.31254886731595,62.366370375273824],[17.31218975291289,62.366280020559756],[17.31218975291289,62.366280020559756],[17.311701858379593,62.366250266747606],[17.31141452118346,62.36621475798874],[17.311298192574387,62.36616829429848],[17.31119551961358,62.3660508732883],[17.31090838678888,62.365941988741156],[17.31090838678888,62.365941988741156],[17.310824104711518,62.36584120050734],[17.3104033473608,62.36558165774728],[17.3104033473608,62.36558165774728],[17.310207842882722,62.365540434014065],[17.309705878005644,62.36547515034254],[17.30956767398036,62.36547813593414],[17.309439434168425,62.36551235403918],[17.309419425558985,62.36551958373221],[17.30921160738498,62.36560576632841],[17.30915579946959,62.365623254415055],[17.30900397864569,62.36567314112198],[17.308851394169547,62.36569830480708],[17.308565107938115,62.36569701301439],[17.308227180085712,62.365765007741764],[17.308096262266435,62.365784853133725],[17.307871340667194,62.36578665213398],[17.307426088606373,62.365708795313985],[17.30683743730231,62.36560013894806],[17.306655943648007,62.36552602329081],[17.306524933653968,62.36538208016681],[17.306373079472305,62.365219813647315],[17.306323305029334,62.36516662584673],[17.306323305029334,62.36516662584673],[17.306266452594866,62.365034676144624],[17.30623650216901,62.36483586874212],[17.306222700110737,62.364736840585614],[17.30615151560317,62.364612284791974],[17.306019130354233,62.364515175726005],[17.305796028683222,62.364430466220014],[17.305517295855044,62.36440039524629],[17.305210794114245,62.36442063741219],[17.304899884059616,62.364483522481656],[17.304246194961618,62.364574830071916],[17.304246194961618,62.364574830071916],[17.303960547892373,62.36461667484179],[17.30374459684939,62.36466380811591],[17.303574151471416,62.36471687789568],[17.303502136130287,62.364745130293635],[17.303445757832637,62.364804472821504],[17.3034454836833,62.3648090489812],[17.303441000521488,62.36488388271841],[17.30345766993305,62.36506007989409],[17.303503686013542,62.36517246979528],[17.30363479781367,62.365287861851215],[17.30436609357238,62.365658379578434],[17.30484009136738,62.36583781784463],[17.305006977650073,62.36589252607683],[17.305228030466434,62.36595310492681],[17.305228030466434,62.36595310492681],[17.30537836882812,62.36599224162196],[17.305508342616786,62.36602471881885],[17.30560852883784,62.36603829715219],[17.30560852883784,62.36603829715219],[17.306496957700258,62.36613481075285],[17.307302228555194,62.3662137303031],[17.307725557731104,62.36625201924858],[17.30823067748755,62.36629122941522],[17.308618659480413,62.3663292735882],[17.308707161238566,62.36635873125322]
			]
		}
	},{"id":1211,"type":"Feature","properties":{"name":"Rännösjöspåret","type":"Skidspår","created":"2019-11-22 12:02:58","updated":"2021-09-08 17:01:19","published":true,"owner":{"organizationID":262,"name":"Matfors Skidklubb"},"manager":{"organizationID":262,"name":"Matfors Skidklubb"},"fields":[{"id":102,"name":"Öppen","type":"TOGGLE","value":"Nej"},{"id":103,"name":"Belysning","type":"TOGGLE","value":"Nej"},{"id":108,"name":"Tillgänglighetsanpassad","type":"TOGGLE","value":"Nej"},{"id":248,"name":"Preparerad skidled klassiskt","type":"TOGGLE","value":"Ja"},{"id":249,"name":"Preparerad skidled skate","type":"TOGGLE","value":"Ja"},{"id":274,"name":"Led","type":"COMBINEDTRAIL","referencedObjects":[{"objectID":2701,"fieldID":262,"direction":"NORMAL"},{"objectID":2702,"fieldID":262,"direction":"NORMAL"},{"objectID":2703,"fieldID":262,"direction":"NORMAL"},{"objectID":2704,"fieldID":262,"direction":"NORMAL"},{"objectID":2705,"fieldID":262,"direction":"NORMAL"},{"objectID":2706,"fieldID":262,"direction":"NORMAL"},{"objectID":2707,"fieldID":262,"direction":"NORMAL"}]},{"id":109,"name":"Svårighet","type":"DROPDOWN","value":"Mycket lätt"},{"id":134,"name":"Ledgrupp","type":"DROPDOWN","value":"Matfors motionsspår"},{"id":104,"name":"Avgift","type":"FREETEXT","value":"Ja"},{"id":110,"name":"Beskrivning","type":"FREETEXT","value":"Skidspår på Rännösjön"},{"id":99,"name":"Längd (meter)","type":"INTEGER","value":6000},{"id":100,"name":"Stigning total (meter)","type":"INTEGER","value":0}]},"geometry":{"type":"LineString","coordinates":[[17.01648047619181,62.34378633813701],[17.015609935142123,62.34342740037375],[17.01484551329004,62.3430173462893],[17.014574834949556,62.34240843020549],[17.014574834949556,62.34240843020549],[17.013388244074267,62.342356470381475],[17.01095934752415,62.34217460993396],[17.00998858757035,62.341811511525655],[17.01014921367259,62.34118064079883],[17.010130317604457,62.34089806658947],[17.0095875607709,62.34061790609238],[17.008833114163146,62.34045595327083],[17.00801428525704,62.34034206711515],[17.006915885189695,62.34009078690073],[17.006002300766934,62.33956969959649],[17.00531045527492,62.33882541774414],[17.004401669745477,62.33736142858847],[17.003269936088845,62.33660776263981],[17.001604317294387,62.335956055057125],[16.999837482593804,62.335305783776455],[16.998831596181862,62.33440885056102],[16.998160247872065,62.333459963185746],[16.998160247872065,62.333459963185746],[16.99667390300223,62.33194137075887],[16.996790175629226,62.331656854937066],[16.99731050364193,62.33135082246451],[16.99791436473729,62.3307764550319],[16.998401829882063,62.330503382287844],[16.998401829882063,62.330503382287844],[16.999152280946063,62.33008297468467],[16.999152280946063,62.33008297468467],[16.999920336806806,62.32994622086378],[17.00120413494007,62.32994346478495],[17.001953748208344,62.33005450065988],[17.00198295429642,62.330020300198264],[17.00219117916848,62.32969674238807],[17.002995118413217,62.32878005400334],[17.003663646324938,62.32846871884256],[17.004386894341632,62.32837029829426],[17.005562818038083,62.32837219896914],[17.006796101275306,62.328423545869356],[17.010052233245982,62.32820680579456],[17.01122046229186,62.32849790314853],[17.012326665503974,62.32887160018939],[17.012552419264857,62.32881176333071],[17.0126942183156,62.32870914275338],[17.012644467517795,62.328571581629255],[17.01233692071144,62.328418895049325],[17.01194962629708,62.32828621827375],[17.0111771471252,62.32805225860304],[17.010261198345386,62.327895795102606],[17.009235490154758,62.327916905845285],[17.00804306224705,62.32807241229296],[17.006794938620317,62.328203570217546],[17.004471178956308,62.328123272941575],[17.003388286058925,62.328189839341704],[17.00294009539257,62.32835971878847],[17.00234450761426,62.32875171144179],[17.00202703584845,62.329019423551166],[17.001786648802543,62.329593737629786],[17.001953748208344,62.33005450065988],[17.001953748208344,62.33005450065988],[17.00198295429642,62.330020300198264],[17.00219117916848,62.32969674238807],[17.002995118413217,62.32878005400334],[17.003663646324938,62.32846871884256],[17.004386894341632,62.32837029829426],[17.005562818038083,62.32837219896914],[17.006796101275306,62.328423545869356],[17.010052233245982,62.32820680579456],[17.01122046229186,62.32849790314853],[17.012326665503974,62.32887160018939],[17.012552419264857,62.32881176333071],[17.0126942183156,62.32870914275338],[17.012644467517795,62.328571581629255],[17.01233692071144,62.328418895049325],[17.01194962629708,62.32828621827375],[17.0111771471252,62.32805225860304],[17.010261198345386,62.327895795102606],[17.009235490154758,62.327916905845285],[17.00804306224705,62.32807241229296],[17.006794938620317,62.328203570217546],[17.004471178956308,62.328123272941575],[17.003388286058925,62.328189839341704],[17.00294009539257,62.32835971878847],[17.00234450761426,62.32875171144179],[17.00202703584845,62.329019423551166],[17.001786648802543,62.329593737629786],[17.001953748208344,62.33005450065988],[17.001953748208344,62.33005450065988],[17.002364403615687,62.33011532660224],[17.003261949331723,62.330400959999984],[17.003840766236895,62.33097402811216],[17.003919505924234,62.331648583572694],[17.003440320568327,62.332064044722756],[17.003374602622262,62.33259925793443],[17.003950326137197,62.33312522929655],[17.005472417592525,62.333653267885126],[17.006807944381066,62.333916849222796],[17.009073075877726,62.33393128524765],[17.01148165398164,62.33406931973484],[17.013152762254357,62.33429655910861],[17.014495746035447,62.3346699560715],[17.01523209743781,62.33506784597976],[17.01543037186478,62.33550495853065],[17.015048366601285,62.335856197057986],[17.014780065760966,62.33603927032959],[17.014780065760966,62.33603927032959],[17.014590410525482,62.33608282445767],[17.01367669873387,62.336064629617326],[17.01202888992735,62.33580563092182],[17.010247708269137,62.33583138839667],[17.009410974916396,62.33595347663282],[17.00892445569878,62.33625906630285],[17.00887685782176,62.33655831491178],[17.009027648576065,62.336791842870426],[17.00890308128699,62.33695077963419],[17.00857173020962,62.33704984783719],[17.00832049007274,62.33733632319357],[17.008580966260677,62.33769397760177],[17.009035649765785,62.33792311647779],[17.009529352586643,62.33823025887215],[17.009811890613783,62.3389175809727],[17.009917208383534,62.339481753736194],[17.01009216536655,62.34007634750091],[17.010305270732857,62.340230404977376],[17.01204095367737,62.340912426439075],[17.012980042331897,62.34130739638377],[17.016652382748706,62.341631331441434],[17.017197961496947,62.34170198916007],[17.017478654152633,62.34185505592336],[17.017387881074765,62.34201351026675],[17.01709764707175,62.34222199849478],[17.0163979522113,62.34237356730732],[17.014574834949556,62.34240843020549]]}}
	]}`

func TestTrailDataLoad(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, response)

	fc := domain.FeatureCollection{}
	json.Unmarshal([]byte(response), &fc)

	err := StoreTrailsFromSource(log.With().Logger(), ctxBrokerMock, context.Background(), server.URL, fc)
	is.NoErr(err)
	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 2)
}

func TestExerciseTrail(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, response)

	ctxBrokerMock.CreateEntityFunc = func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
		return &ngsild.CreateEntityResult{}, nil
	}

	client := NewClient("apiKey", server.URL, zerolog.Logger{})

	featureCollection, err := client.Get(context.Background())
	is.NoErr(err)

	err = StoreTrailsFromSource(zerolog.Logger{}, ctxBrokerMock, context.Background(), server.URL, *featureCollection)
	is.NoErr(err)

	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 2)
	e := ctxBrokerMock.CreateEntityCalls()[0].Entity
	entityJSON, _ := json.Marshal(e)

	const difficulty string = `"difficulty":{"type":"Property","value":0.5}`
	const payment string = `"paymentRequired":{"type":"Property","value":"yes"}`

	is.True(strings.Contains(string(entityJSON), difficulty))
	is.True(strings.Contains(string(entityJSON), payment))
}

func TestExerciseTrailContainsManagerAndOwnerProperties(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, response)

	ctxBrokerMock.CreateEntityFunc = func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
		return &ngsild.CreateEntityResult{}, nil
	}

	client := NewClient("apiKey", server.URL, zerolog.Logger{})

	featureCollection, err := client.Get(context.Background())
	is.NoErr(err)

	err = StoreTrailsFromSource(zerolog.Logger{}, ctxBrokerMock, context.Background(), server.URL, *featureCollection)
	is.NoErr(err)

	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 2)
	e := ctxBrokerMock.CreateEntityCalls()[0].Entity
	entityJSON, _ := json.Marshal(e)

	const manager string = `"manager":{"type":"Relationship","object":"urn:ngsi-ld:Organisation:se:sundsvall:88"}`
	const owner string = `"owner":{"type":"Relationship","object":"urn:ngsi-ld:Organisation:se:sundsvall:36"}`
	is.True(strings.Contains(string(entityJSON), manager))
	is.True(strings.Contains(string(entityJSON), owner))
}

func setupMockServiceThatReturns(is *is.I, expectedRequestBody string, responseCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expectedRequestBody != "" {
			bodyBytes, err := io.ReadAll(r.Body)
			is.NoErr(err)
			defer r.Body.Close()

			is.Equal(string(bodyBytes), expectedRequestBody)
		}
		w.WriteHeader(responseCode)
		w.Header().Add("Content-Type", "application/json")
		if body != "" {
			w.Write([]byte(body))
		}
	}))
}

func testSetup(t *testing.T, requestBody string, responseCode int, responseBody string) (*is.I, *test.ContextBrokerClientMock, *httptest.Server) {
	is := is.New(t)
	mockServer := setupMockServiceThatReturns(is, requestBody, responseCode, responseBody)
	ctxBroker := &test.ContextBrokerClientMock{
		CreateEntityFunc: func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
			return nil, fmt.Errorf("not implemented")
		},
		MergeEntityFunc: func(ctx context.Context, entityID string, fragment types.EntityFragment, headers map[string][]string) (*ngsild.MergeEntityResult, error) {
			return nil, ngsierrors.ErrNotFound
		},
	}

	return is, ctxBroker, mockServer
}
